//! The Hub API client of the Rust CLI, on ureq (blocking, rustls with
//! `ring`): the Go CLI's URL rules, TLS trust, proxy rules, same-origin
//! redirects, size limit and timeout. Error texts are plain words of our
//! own; the exit code comes from the error's type.
#![forbid(unsafe_code)]

mod envelope;
pub mod error;
pub mod escape;
pub mod gofloat;
pub mod gojson;
pub mod proxy;
pub mod redact;
mod request;
pub mod tls;
mod transport;

use std::io::{Read, Write};
use std::time::Duration;

use serde_json::Value;
use url::Url;

pub use envelope::decode as decode_envelope;
pub use error::ClientError;
pub use proxy::ProxyEnv;
pub use request::Method;

/// The largest Hub answer the CLI reads.
pub const MAX_RESPONSE_BYTES: u64 = 16 * 1024 * 1024;
/// The whole request, redirects included, must finish within this time.
pub const DEFAULT_TIMEOUT: Duration = Duration::from_secs(5 * 60);
/// The largest file `files cat` prints.
pub const MAX_DOWNLOAD_BYTES: u64 = 512 * 1024 * 1024;
/// Redirects followed before the CLI stops.
pub const MAX_REDIRECTS: usize = 10;

/// An API key. It has no `Display`, and `Debug` never shows it, so it cannot
/// reach an error message or a log by accident.
#[derive(Clone)]
pub struct Secret(String);

impl Secret {
    pub fn new(value: impl Into<String>) -> Self {
        Secret(value.into())
    }
    pub fn expose(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Debug for Secret {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("Secret([redacted])")
    }
}

/// Go's `isLoopbackHost`.
pub fn is_loopback_host(host: &str) -> bool {
    let host = host
        .trim()
        .trim_start_matches('[')
        .trim_end_matches(']')
        .to_ascii_lowercase();
    let host = host.strip_suffix('.').unwrap_or(&host);
    if host == "localhost" {
        return true;
    }
    match host.parse::<std::net::IpAddr>() {
        Ok(std::net::IpAddr::V4(ip)) => ip.octets()[0] == 127,
        Ok(std::net::IpAddr::V6(ip)) => {
            ip.is_loopback() || ip.to_ipv4_mapped().is_some_and(|v4| v4.octets()[0] == 127)
        }
        Err(_) => false,
    }
}

/// The Go CLI's `ValidateBaseURL` rules, with the `url` crate parsing:
/// HTTPS, or plain HTTP to a loopback address only; no credentials, query
/// or fragment in the URL. The messages are the Go CLI's.
pub fn validate_base_url(raw: &str) -> Result<(), String> {
    let parsed = Url::parse(raw.trim()).map_err(|_| "invalid hub URL".to_string())?;
    let host = parsed.host_str().unwrap_or_default();
    if host.is_empty() || !parsed.username().is_empty() || parsed.password().is_some() {
        return Err("invalid hub URL".to_string());
    }
    if parsed.query().is_some() || parsed.fragment().is_some() {
        return Err("hub URL must not include a query or fragment".to_string());
    }
    match parsed.scheme() {
        "https" => Ok(()),
        "http" if is_loopback_host(host) => Ok(()),
        "http" => Err(format!(
            "refusing to send API credentials over plaintext HTTP to non-loopback host \"{}\"",
            host.trim_start_matches('[').trim_end_matches(']')
        )),
        _ => Err("hub URL must use HTTPS (HTTP is allowed only for loopback)".to_string()),
    }
}

/// The URL text for messages: any password replaced, as Go's `Redacted`.
fn shown(url: &Url) -> String {
    let mut copy = url.clone();
    if copy.password().is_some() {
        let _ = copy.set_password(Some("xxxxx"));
    }
    copy.to_string()
}

fn same_origin(a: &Url, b: &Url) -> bool {
    a.scheme() == b.scheme() && a.host_str() == b.host_str() && a.port() == b.port()
}

pub struct Client {
    pub(crate) base_url: String,
    pub(crate) api_key: Secret,
    pub(crate) roots: ureq::tls::RootCerts,
    pub(crate) proxy: ProxyEnv,
    /// The deadline for one request, redirects included; tests shorten it.
    pub timeout: Duration,
}

impl Client {
    /// Builds a client. A bad CA file is a settings error.
    pub fn new(
        base_url: &str,
        api_key: Secret,
        ca_file: Option<&str>,
        proxy: ProxyEnv,
    ) -> Result<Self, ClientError> {
        let ca_file = ca_file.filter(|path| !path.trim().is_empty());
        Ok(Client {
            base_url: base_url.trim().trim_end_matches('/').to_string(),
            api_key: Secret::new(api_key.expose().trim()),
            roots: tls::root_certs(ca_file)?,
            proxy,
            timeout: DEFAULT_TIMEOUT,
        })
    }

    /// `GET path` and the decoded `data` payload.
    pub fn get(&self, path: &str) -> Result<Value, ClientError> {
        let (status, body) = self.fetch(path)?;
        envelope::decode(status, &body)
    }

    /// `method path` with an optional JSON body (POST and PATCH), and the
    /// decoded `data` payload.
    pub fn send(
        &self,
        method: Method,
        path: &str,
        body: Option<&[u8]>,
    ) -> Result<Value, ClientError> {
        let response = self.open(method, path, body)?;
        let (status, body) = read_limited(response)?;
        envelope::decode(status, &body)
    }

    /// `GET path` with same-origin redirects followed: the final status and
    /// body (at most 16 MiB).
    pub fn fetch(&self, path: &str) -> Result<(u16, Vec<u8>), ClientError> {
        read_limited(self.open(Method::Get, path, None)?)
    }

    /// The Go CLI's `Download`: a file body that is not a JSON envelope,
    /// written to `out` as it arrives, at most 512 MiB. An error status
    /// gives the Hub's message, or "request failed (status N)".
    pub fn download(&self, path: &str, out: &mut dyn Write) -> Result<(), ClientError> {
        let mut response = self.open(Method::Get, path, None)?;
        let status = response.status().as_u16();
        if !(200..300).contains(&status) {
            let (_, body) = read_limited(response)?;
            return Err(envelope::download_error(status, &body));
        }
        let declared = response
            .headers()
            .get("content-length")
            .and_then(|v| v.to_str().ok())
            .and_then(|v| v.trim().parse::<u64>().ok());
        if declared.is_some_and(|length| length > MAX_DOWNLOAD_BYTES) {
            return Err(too_large(MAX_DOWNLOAD_BYTES));
        }
        let mut reader = response.body_mut().with_config().limit(u64::MAX).reader();
        let read_failed = |e: std::io::Error| {
            ClientError::Failed(format!("read response: {}", transport::io_text(&e)))
        };
        let mut written: u64 = 0;
        let mut buffer = vec![0u8; 64 * 1024];
        while written < MAX_DOWNLOAD_BYTES {
            let room = (MAX_DOWNLOAD_BYTES - written).min(buffer.len() as u64) as usize;
            let n = match reader.read(&mut buffer[..room]) {
                Ok(0) => return Ok(()),
                Ok(n) => n,
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => continue,
                Err(e) => return Err(read_failed(e)),
            };
            out.write_all(&buffer[..n]).map_err(|e| {
                ClientError::Failed(format!("write output: {}", error::errno_text(&e)))
            })?;
            written += n as u64;
        }
        // Exactly the limit was written: one more byte means the file is
        // larger, as the Go CLI checks.
        let mut extra = [0u8; 1];
        match reader.read(&mut extra) {
            Ok(0) => Ok(()),
            Ok(_) => Err(too_large(MAX_DOWNLOAD_BYTES)),
            Err(e) => Err(read_failed(e)),
        }
    }
}

fn too_large(limit: u64) -> ClientError {
    ClientError::Failed(format!("response exceeds {limit} byte limit"))
}

/// Reads an answer's body, at most 16 MiB.
fn read_limited(
    mut response: ureq::http::Response<ureq::Body>,
) -> Result<(u16, Vec<u8>), ClientError> {
    let status = response.status().as_u16();
    match response
        .body_mut()
        .with_config()
        .limit(MAX_RESPONSE_BYTES)
        .read_to_vec()
    {
        Ok(body) => Ok((status, body)),
        Err(ureq::Error::BodyExceedsLimit(_)) => Err(too_large(MAX_RESPONSE_BYTES)),
        Err(e) => Err(ClientError::Failed(format!(
            "read response: {}",
            transport::plain(&e)
        ))),
    }
}

#[cfg(test)]
mod tests {
    use super::validate_base_url;

    // Fault: the Rust CLI accepted a Hub URL the Go CLI refuses, so an API
    // key could go over plain HTTP to another machine, or with credentials
    // or a query in the URL.
    #[test]
    fn base_url_rules_match_go() {
        for ok in [
            "https://nas01:8443",
            "http://127.0.0.1:8080",
            "http://localhost",
            "http://[::1]:80",
            "https://nas01/prefix",
        ] {
            assert_eq!(validate_base_url(ok), Ok(()), "{ok}");
        }
        assert_eq!(
            validate_base_url("http://192.0.2.10:8443"),
            Err("refusing to send API credentials over plaintext HTTP to non-loopback host \"192.0.2.10\"".into())
        );
        for bad in [
            "not a url",
            "https://user:pw@nas01",
            "https://user@nas01",
            "https://nas 01",
        ] {
            assert_eq!(
                validate_base_url(bad),
                Err("invalid hub URL".into()),
                "{bad}"
            );
        }
        assert_eq!(
            validate_base_url("https://nas01?a=1"),
            Err("hub URL must not include a query or fragment".into())
        );
        assert_eq!(
            validate_base_url("https://nas01#x"),
            Err("hub URL must not include a query or fragment".into())
        );
        assert_eq!(
            validate_base_url("ftp://nas01"),
            Err("hub URL must use HTTPS (HTTP is allowed only for loopback)".into())
        );
    }
}
