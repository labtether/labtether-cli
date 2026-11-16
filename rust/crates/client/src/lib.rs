//! The Hub API client of the Rust CLI, on ureq (blocking, rustls with
//! `ring`): the Go CLI's URL rules, TLS trust, proxy rules, same-origin
//! redirects, size limit and timeout. Error texts are plain words of our
//! own; the exit code comes from the error's type.
#![forbid(unsafe_code)]

mod envelope;
pub mod error;
pub mod escape;
pub mod proxy;
pub mod tls;
mod transport;

use std::time::{Duration, Instant};

use serde_json::Value;
use url::Url;

pub use envelope::decode as decode_envelope;
pub use error::ClientError;
pub use proxy::ProxyEnv;

/// The largest Hub answer the CLI reads.
pub const MAX_RESPONSE_BYTES: u64 = 16 * 1024 * 1024;
/// The whole request, redirects included, must finish within this time.
pub const DEFAULT_TIMEOUT: Duration = Duration::from_secs(5 * 60);
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
    base_url: String,
    api_key: Secret,
    roots: ureq::tls::RootCerts,
    proxy: ProxyEnv,
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

    /// `GET path` with same-origin redirects followed: the final status and
    /// body (at most 16 MiB).
    pub fn fetch(&self, path: &str) -> Result<(u16, Vec<u8>), ClientError> {
        validate_base_url(&self.base_url).map_err(ClientError::Failed)?;
        let origin = Url::parse(&self.base_url)
            .map_err(|_| ClientError::Failed("invalid hub URL".into()))?;
        let mut current = Url::parse(&format!("{}{path}", self.base_url)).map_err(|_| {
            ClientError::Failed(format!("create request: invalid URL {:?}", shown(&origin)))
        })?;
        let deadline = Instant::now() + self.timeout;
        let mut previous: Option<Url> = None;
        for hop in 0..=MAX_REDIRECTS + 1 {
            let failed = |message: String| {
                ClientError::Failed(format!(
                    "request failed: Get \"{}\": {message}",
                    shown(&current)
                ))
            };
            let left = deadline.saturating_duration_since(Instant::now());
            if left.is_zero() {
                return Err(failed("timed out".into()));
            }
            let agent =
                transport::agent(&self.roots, &self.proxy, &current, left).map_err(failed)?;
            let mut request = agent
                .get(current.as_str())
                .header("Authorization", format!("Bearer {}", self.api_key.expose()));
            if let Some(referer) = previous
                .as_ref()
                .filter(|last| !(last.scheme() == "https" && current.scheme() == "http"))
            {
                request = request.header("Referer", shown(referer));
            }
            let mut response = request.call().map_err(|e| failed(transport::plain(&e)))?;
            let status = response.status().as_u16();
            let location = response
                .headers()
                .get("location")
                .and_then(|v| v.to_str().ok())
                .map(str::to_string);
            match location
                .filter(|l| !l.is_empty() && matches!(status, 301 | 302 | 303 | 307 | 308))
            {
                None => {
                    let body = response
                        .body_mut()
                        .with_config()
                        .limit(MAX_RESPONSE_BYTES)
                        .read_to_vec();
                    return match body {
                        Ok(body) => Ok((status, body)),
                        Err(ureq::Error::BodyExceedsLimit(_)) => Err(ClientError::Failed(format!(
                            "response exceeds {MAX_RESPONSE_BYTES} byte limit"
                        ))),
                        Err(e) => Err(ClientError::Failed(format!(
                            "read response: {}",
                            transport::plain(&e)
                        ))),
                    };
                }
                Some(location) => {
                    let refused = |message: String| {
                        ClientError::Failed(format!(
                            "request failed: Get \"{location}\": {message}"
                        ))
                    };
                    let next = current.join(&location).map_err(|_| {
                        failed(format!(
                            "the hub sent an unreadable redirect location {location:?}"
                        ))
                    })?;
                    // Go's client checks before following: with 10 requests
                    // sent it refuses the 10th redirect.
                    if hop + 1 >= MAX_REDIRECTS {
                        return Err(refused(format!("stopped after {MAX_REDIRECTS} redirects")));
                    }
                    if !same_origin(&origin, &next) {
                        return Err(refused(format!(
                            "refusing cross-origin redirect to {}",
                            shown(&next)
                        )));
                    }
                    previous = Some(std::mem::replace(&mut current, next));
                }
            }
        }
        unreachable!("the loop returns within MAX_REDIRECTS + 1 hops")
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
