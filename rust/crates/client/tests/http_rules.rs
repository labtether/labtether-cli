//! The client's transport rules against a stand-in Hub: redirects, size
//! limit, timeout, connection errors and proxy choice.

mod common;

use std::time::Duration;

use common::{ok_json, redirect, serve};
use labtether_cli_client::{Client, ClientError, MAX_RESPONSE_BYTES, ProxyEnv, Secret};

fn client(port: u16) -> Client {
    Client::new(
        &format!("http://127.0.0.1:{port}"),
        Secret::new("test-key"),
        None,
        ProxyEnv::default(),
    )
    .unwrap()
}

// Fault: a same-origin redirect was not followed, or the API key was not
// sent on the next hop (the Go CLI keeps it within the same origin).
#[test]
fn same_origin_redirect_is_followed_with_the_key() {
    let hub = serve(Box::new(|line, _| {
        if line.starts_with("GET /old ") {
            redirect("/api/v2/new?x=1")
        } else {
            ok_json(r#"{"data":{"ok":true}}"#)
        }
    }));
    let data = client(hub.port).get("/old").unwrap();
    assert!(data.get("ok").is_some());
    let requests = hub.requests.lock().unwrap();
    assert_eq!(requests.len(), 2);
    assert!(requests[1].starts_with("GET /api/v2/new?x=1 HTTP/1.1\r\n"));
    assert!(
        requests[1]
            .to_ascii_lowercase()
            .contains("authorization: bearer test-key\r\n")
    );
    assert!(
        requests[1]
            .to_ascii_lowercase()
            .contains(&format!("referer: http://127.0.0.1:{}/old\r\n", hub.port))
    );
}

// Fault: a redirect to another host or port was followed, which would send
// the API key there.
#[test]
fn cross_origin_redirect_is_refused() {
    let hub = serve(Box::new(|_, _| redirect("http://127.0.0.1:1/steal")));
    let err = client(hub.port).get("/api/v2/assets").unwrap_err();
    assert_eq!(err, ClientError::Failed(
        "request failed: Get \"http://127.0.0.1:1/steal\": refusing cross-origin redirect to http://127.0.0.1:1/steal".into()
    ));
    assert_eq!(hub.requests.lock().unwrap().len(), 1);
}

// Fault: a redirect loop was followed forever or stopped at another count
// than Go's (10 requests, then "stopped after 10 redirects"; found by the
// parity run).
#[test]
fn redirects_stop_after_ten() {
    let hub = serve(Box::new(|line, _| {
        let n: u32 = line
            .split('/')
            .nth(1)
            .and_then(|p| p.split(' ').next())
            .and_then(|p| p.parse().ok())
            .unwrap_or(0);
        redirect(&format!("/{}", n + 1))
    }));
    let err = client(hub.port).get("/0").unwrap_err();
    assert_eq!(
        err.to_string(),
        "request failed: Get \"/10\": stopped after 10 redirects"
    );
    assert_eq!(hub.requests.lock().unwrap().len(), 10);
}

// Fault: an oversized answer was read into memory without a limit.
#[test]
fn responses_over_16_mib_are_refused() {
    let hub = serve(Box::new(|line, _| {
        let size = if line.contains("/big") {
            MAX_RESPONSE_BYTES as usize + 1
        } else {
            MAX_RESPONSE_BYTES as usize
        };
        let mut out = format!("HTTP/1.1 200 OK\r\nContent-Length: {size}\r\n\r\n").into_bytes();
        out.resize(out.len() + size, b' ');
        out
    }));
    let err = client(hub.port).fetch("/big").unwrap_err();
    assert_eq!(err.to_string(), "response exceeds 16777216 byte limit");
    let (status, body) = client(hub.port).fetch("/exact").unwrap();
    assert_eq!((status, body.len() as u64), (200, MAX_RESPONSE_BYTES));
}

// Fault: a Hub that accepts the connection and never answers hung the CLI.
#[test]
fn a_silent_hub_times_out() {
    let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    let holder = std::thread::spawn(move || {
        let conn = listener.accept();
        std::thread::sleep(Duration::from_secs(2));
        drop(conn);
    });
    let mut c = client(port);
    c.timeout = Duration::from_millis(300);
    let started = std::time::Instant::now();
    let err = c.get("/api/v2/whoami").unwrap_err();
    assert!(
        started.elapsed() < Duration::from_millis(1500),
        "took {:?}",
        started.elapsed()
    );
    assert_eq!(
        err.to_string(),
        format!("request failed: Get \"http://127.0.0.1:{port}/api/v2/whoami\": timed out")
    );
    holder.join().unwrap();
}

#[test]
fn refused_connection_uses_go_text() {
    let port = {
        let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
        listener.local_addr().unwrap().port()
    };
    let err = client(port).get("/api/v2/whoami").unwrap_err();
    assert_eq!(
        err.to_string(),
        format!(
            "request failed: Get \"http://127.0.0.1:{port}/api/v2/whoami\": connection refused"
        )
    );
}

// Fault: a body cut short by the Hub was accepted as complete.
#[test]
fn short_body_is_an_error() {
    let hub = serve(Box::new(|_, _| {
        b"HTTP/1.1 200 OK\r\nContent-Length: 50\r\n\r\n{\"data\":".to_vec()
    }));
    let text = client(hub.port).get("/x").unwrap_err().to_string();
    assert!(text.starts_with("read response: "), "{text}");
}

#[test]
fn chunked_bodies_are_decoded() {
    let hub = serve(Box::new(|_, _| {
        b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\n{\"dat\r\na;x=1\r\na\":[1,2]}\n\r\n0\r\n\r\n".to_vec()
    }));
    assert_eq!(client(hub.port).get("/x").unwrap().to_string(), "[1,2]");
}

/// A stand-in proxy that records the first request line and refuses it.
fn proxy() -> (u16, std::sync::Arc<std::sync::Mutex<Vec<String>>>) {
    let stand_in = serve(Box::new(|_, _| {
        b"HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n".to_vec()
    }));
    (stand_in.port, stand_in.requests)
}

// Fault: the Rust CLI ignored HTTPS_PROXY, which the Go CLI honours
// through its transport, so it could not reach a Hub behind a proxy.
#[test]
fn https_proxy_is_used_for_a_remote_hub() {
    let (port, seen) = proxy();
    let env = ProxyEnv {
        https: Some(format!("http://127.0.0.1:{port}")),
        ..ProxyEnv::default()
    };
    let mut c = Client::new(
        "https://hub.example:8443",
        Secret::new("test-key"),
        None,
        env,
    )
    .unwrap();
    c.timeout = Duration::from_secs(5);
    assert!(c.get("/api/v2/whoami").is_err());
    let seen = seen.lock().unwrap();
    assert!(
        seen.first()
            .is_some_and(|r| r.starts_with("CONNECT hub.example:8443 ")),
        "{seen:?}"
    );
    assert!(
        !seen
            .iter()
            .any(|r| r.to_ascii_lowercase().contains("test-key")),
        "the API key reached the proxy"
    );
}

// Fault: a Hub named in NO_PROXY, or on a loopback address, went through
// the proxy (Go connects directly in both cases).
#[test]
fn no_proxy_and_loopback_connect_directly() {
    let (port, seen) = proxy();
    let hub = serve(Box::new(|_, _| ok_json(r#"{"data":{"ok":true}}"#)));
    let address = format!("http://127.0.0.1:{port}");
    let env = ProxyEnv {
        http: Some(address.clone()),
        https: Some(address),
        no_proxy: "hub.example".into(),
    };
    let direct = Client::new(
        &format!("http://127.0.0.1:{}", hub.port),
        Secret::new("k"),
        None,
        env.clone(),
    )
    .unwrap();
    assert!(direct.get("/x").is_ok());
    let mut named = Client::new("https://hub.example:8443", Secret::new("k"), None, env).unwrap();
    named.timeout = Duration::from_secs(5);
    assert!(named.get("/x").is_err());
    assert!(
        seen.lock().unwrap().is_empty(),
        "proxy was used: {:?}",
        seen.lock().unwrap()
    );
}
