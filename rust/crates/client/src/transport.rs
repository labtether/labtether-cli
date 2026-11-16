//! The ureq agent for one request and plain error texts for its failures.

use std::time::Duration;

use ureq::tls::{RootCerts, TlsConfig};
use ureq::{Agent, Proxy};
use url::Url;

use crate::proxy::ProxyEnv;

/// Go's transport gives a TLS handshake 10 s (`TLSHandshakeTimeout`). ureq
/// runs the handshake inside its connect step, so the connect step gets
/// that limit; without it a Hub port that accepts and never answers held
/// the CLI for the whole 5 minute request limit.
pub const CONNECT_TIMEOUT: Duration = Duration::from_secs(10);

/// An agent that follows no redirect (the client checks each one against
/// the Hub's origin), returns 4xx and 5xx answers as answers, trusts
/// `roots`, uses the proxy Go would use for `url`, and gives up after
/// `timeout`. Connecting, proxy setup and the TLS handshake together get
/// at most `CONNECT_TIMEOUT`.
pub fn agent(
    roots: &RootCerts,
    proxies: &ProxyEnv,
    url: &Url,
    timeout: Duration,
) -> Result<Agent, String> {
    let host = url.host_str().unwrap_or_default();
    let port = url.port_or_known_default().unwrap_or(443);
    let proxy = match proxies.for_url(url.scheme(), host, port) {
        Some(address) => Some(
            Proxy::new(&address).map_err(|_| "the proxy setting is not a valid URL".to_string())?,
        ),
        None => None,
    };
    let config = Agent::config_builder()
        .http_status_as_error(false)
        .max_redirects(0)
        .timeout_global(Some(timeout))
        .timeout_connect(Some(CONNECT_TIMEOUT))
        .proxy(proxy)
        .tls_config(TlsConfig::builder().root_certs(roots.clone()).build())
        .build();
    Ok(config.into())
}

/// Plain words for a TLS failure.
fn tls_text(err: &rustls::Error) -> String {
    use rustls::{CertificateError as C, Error as Tls};
    match err {
        // BadSignature: a trusted CA has the issuer's name but did not sign it.
        Tls::InvalidCertificate(C::UnknownIssuer | C::BadSignature) => {
            "TLS certificate signed by an unknown authority".into()
        }
        Tls::InvalidCertificate(C::NotValidForName | C::NotValidForNameContext { .. }) => {
            "TLS certificate is not valid for this host name".into()
        }
        Tls::InvalidCertificate(C::Expired | C::ExpiredContext { .. }) => {
            "TLS certificate has expired".into()
        }
        other => format!("TLS failed: {other}"),
    }
}

/// Plain words for a failed request. These texts are the Rust CLI's own;
/// the exit code (1) is the Go CLI's.
pub fn plain(err: &ureq::Error) -> String {
    use ureq::Error;
    match err {
        Error::Io(io) => {
            if let Some(tls) = io
                .get_ref()
                .and_then(|inner| inner.downcast_ref::<rustls::Error>())
            {
                return tls_text(tls);
            }
            match io.kind() {
                std::io::ErrorKind::ConnectionRefused => "connection refused".into(),
                std::io::ErrorKind::ConnectionReset => "the connection was reset".into(),
                std::io::ErrorKind::TimedOut => "timed out".into(),
                std::io::ErrorKind::UnexpectedEof => {
                    "the connection closed before the answer was complete".into()
                }
                _ => crate::error::errno_text(io),
            }
        }
        Error::HostNotFound => "host not found".into(),
        Error::ConnectionFailed => "connection failed".into(),
        Error::Timeout(_) => "timed out".into(),
        Error::Rustls(tls) => tls_text(tls),
        Error::Tls(text) => format!("TLS failed: {text}"),
        Error::ConnectProxyFailed(_) => "the proxy refused the connection".into(),
        Error::InvalidProxyUrl => "the proxy setting is not a valid URL".into(),
        Error::Protocol(_) => "the answer is not valid HTTP".into(),
        Error::LargeResponseHeader(_, _) => "the answer's headers are too large".into(),
        other => format!("{other}"),
    }
}
