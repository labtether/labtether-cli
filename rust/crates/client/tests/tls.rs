//! TLS trust: the private CA file is added to the system roots, a Hub
//! signed by an unknown CA is refused with Go's message, and a plain HTTP
//! answer to TLS is named.

mod common;

use std::io::{Read, Write};
use std::net::TcpListener;
use std::sync::Arc;

use labtether_cli_client::{Client, ClientError, ProxyEnv, Secret};
use rcgen::{BasicConstraints, CertificateParams, CertifiedIssuer, IsCa, KeyPair};
use rustls::pki_types::{PrivateKeyDer, PrivatePkcs8KeyDer};

struct Pki {
    ca_pem: String,
    config: Arc<rustls::ServerConfig>,
}

fn pki() -> Pki {
    let mut ca_params = CertificateParams::new(Vec::<String>::new()).unwrap();
    ca_params.is_ca = IsCa::Ca(BasicConstraints::Unconstrained);
    let ca = CertifiedIssuer::self_signed(ca_params, KeyPair::generate().unwrap()).unwrap();
    let key = KeyPair::generate().unwrap();
    let cert = CertificateParams::new(vec!["127.0.0.1".to_owned()])
        .unwrap()
        .signed_by(&key, &ca)
        .unwrap();
    let provider = Arc::new(rustls::crypto::ring::default_provider());
    let config = rustls::ServerConfig::builder_with_provider(provider)
        .with_protocol_versions(&[&rustls::version::TLS12])
        .unwrap()
        .with_no_client_auth()
        .with_single_cert(
            vec![cert.der().clone()],
            PrivateKeyDer::Pkcs8(PrivatePkcs8KeyDer::from(key.serialize_der())),
        )
        .unwrap();
    Pki {
        ca_pem: ca.pem(),
        config: Arc::new(config),
    }
}

/// A TLS 1.2-only Hub answering one JSON body per connection.
fn tls_hub(config: Arc<rustls::ServerConfig>) -> u16 {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    std::thread::spawn(move || {
        for stream in listener.incoming() {
            let Ok(tcp) = stream else { break };
            let conn = rustls::ServerConnection::new(config.clone()).unwrap();
            let mut tls = rustls::StreamOwned::new(conn, tcp);
            let mut buf = [0u8; 4096];
            if tls.read(&mut buf).is_err() {
                continue;
            }
            let body = r#"{"data":{"secure":true}}"#;
            let _ = tls.write_all(
                format!(
                    "HTTP/1.1 200 OK\r\nContent-Length: {}\r\n\r\n{body}",
                    body.len()
                )
                .as_bytes(),
            );
            tls.conn.send_close_notify();
            let _ = tls.flush();
        }
    });
    port
}

fn write_temp(name: &str, text: &str) -> String {
    let path = std::env::temp_dir().join(format!("ltcli-tls-{name}-{}", std::process::id()));
    std::fs::write(&path, text).unwrap();
    path.to_string_lossy().into_owned()
}

// Fault: the private CA file was not trusted (the Hub's default setup
// failed), or a Hub signed by an unknown CA was accepted. The Hub here
// speaks only TLS 1.2, which the CLI must still accept.
#[test]
fn ca_file_is_trusted_and_unknown_ca_is_refused() {
    let hub = pki();
    let port = tls_hub(hub.config.clone());
    let base = format!("https://127.0.0.1:{port}");
    let ca = write_temp("ca", &hub.ca_pem);
    let trusted = Client::new(&base, Secret::new("k"), Some(&ca), ProxyEnv::default()).unwrap();
    assert!(
        trusted
            .get("/api/v2/whoami")
            .unwrap()
            .get("secure")
            .is_some()
    );
    let other = write_temp("other", &pki().ca_pem);
    let untrusted =
        Client::new(&base, Secret::new("k"), Some(&other), ProxyEnv::default()).unwrap();
    assert_eq!(
        untrusted.get("/api/v2/whoami").unwrap_err().to_string(),
        format!(
            "request failed: Get \"{base}/api/v2/whoami\": TLS certificate signed by an unknown authority"
        )
    );
}

// Fault: a CA file with no certificate was accepted silently.
#[test]
fn ca_file_without_certificates_is_a_settings_error() {
    let bad = write_temp("bad", "not a certificate\n");
    let err = Client::new(
        "https://127.0.0.1:1",
        Secret::new("k"),
        Some(&bad),
        ProxyEnv::default(),
    )
    .err()
    .unwrap();
    assert_eq!(
        err,
        ClientError::Config("TLS CA file contains no valid PEM certificates".into())
    );
}

// Fault: a plain HTTP answer on the TLS port hung or was read as a Hub
// answer (the API key would then have gone over plain text).
#[test]
fn plain_http_answer_to_tls_fails() {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    std::thread::spawn(move || {
        if let Ok((mut tcp, _)) = listener.accept() {
            let mut buf = [0u8; 512];
            let _ = tcp.read(&mut buf);
            let _ = tcp.write_all(&common::ok_json("{}"));
        }
    });
    let base = format!("https://127.0.0.1:{port}");
    let mut client = Client::new(&base, Secret::new("k"), None, ProxyEnv::default()).unwrap();
    client.timeout = std::time::Duration::from_secs(5);
    let err = client.get("/x").unwrap_err();
    let text = err.to_string();
    assert!(
        matches!(err, ClientError::Failed(_))
            && text.starts_with(&format!("request failed: Get \"{base}/x\": ")),
        "{text}"
    );
}

// Fault: a Hub port that accepts and never sends a TLS answer held the CLI
// for the whole 5 minute request limit; Go's transport gives the TLS
// handshake 10 s.
#[test]
fn a_silent_tls_port_gives_up_after_the_handshake_limit() {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    std::thread::spawn(move || {
        let held = listener.accept();
        std::thread::sleep(std::time::Duration::from_secs(40));
        drop(held);
    });
    let base = format!("https://127.0.0.1:{port}");
    let client = Client::new(&base, Secret::new("k"), None, ProxyEnv::default()).unwrap();
    let started = std::time::Instant::now();
    let err = client.get("/x").unwrap_err();
    let took = started.elapsed();
    assert!(
        took >= std::time::Duration::from_secs(9) && took < std::time::Duration::from_secs(20),
        "took {took:?}"
    );
    assert_eq!(
        err.to_string(),
        format!("request failed: Get \"{base}/x\": timed out")
    );
}
