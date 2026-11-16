//! Which proxy a request uses, by Go's `http.ProxyFromEnvironment` rules
//! (the Go CLI clones `http.DefaultTransport`): `HTTPS_PROXY` for https
//! URLs and `HTTP_PROXY` for http URLs (upper case first, then lower case),
//! never for localhost or a loopback address, and not for hosts matched by
//! `NO_PROXY`. ureq's own `Proxy::try_from_env` picks one proxy for every
//! scheme, so the choice is made here and the URL handed to ureq.

use std::net::IpAddr;

/// The proxy settings, read once from the environment.
#[derive(Clone, Debug, Default)]
pub struct ProxyEnv {
    pub http: Option<String>,
    pub https: Option<String>,
    pub no_proxy: String,
}

fn any(get: &dyn Fn(&str) -> Option<String>, names: &[&str]) -> Option<String> {
    names
        .iter()
        .find_map(|name| get(name).filter(|v| !v.is_empty()))
}

impl ProxyEnv {
    pub fn from_process() -> ProxyEnv {
        Self::from_lookup(&|name| std::env::var(name).ok())
    }

    pub fn from_lookup(get: &dyn Fn(&str) -> Option<String>) -> ProxyEnv {
        ProxyEnv {
            http: any(get, &["HTTP_PROXY", "http_proxy"]),
            https: any(get, &["HTTPS_PROXY", "https_proxy"]),
            no_proxy: any(get, &["NO_PROXY", "no_proxy"]).unwrap_or_default(),
        }
    }

    /// The proxy URL for a request, or `None` to connect directly. A proxy
    /// written without a scheme means `http://`, as in Go.
    pub fn for_url(&self, scheme: &str, host: &str, port: u16) -> Option<String> {
        let chosen = if scheme == "https" {
            self.https.as_ref()
        } else {
            self.http.as_ref()
        }?;
        if !self.use_proxy(host, port) {
            return None;
        }
        Some(if chosen.contains("://") {
            chosen.clone()
        } else {
            format!("http://{chosen}")
        })
    }

    fn use_proxy(&self, host: &str, port: u16) -> bool {
        let host = host
            .trim_start_matches('[')
            .trim_end_matches(']')
            .to_ascii_lowercase();
        if host == "localhost" {
            return false;
        }
        let ip = host.parse::<IpAddr>().ok();
        if ip.is_some_and(|ip| ip.is_loopback() || matches!(ip, IpAddr::V6(v6) if v6.to_ipv4_mapped().is_some_and(|v4| v4.is_loopback()))) {
            return false;
        }
        for entry in self
            .no_proxy
            .split(',')
            .map(|e| e.trim().to_ascii_lowercase())
            .filter(|e| !e.is_empty())
        {
            if entry == "*" {
                return false;
            }
            if let (Some((net, bits)), Some(ip)) = (entry.split_once('/'), ip)
                && let (Ok(net), Ok(bits)) = (net.parse::<IpAddr>(), bits.parse::<u32>())
                && in_cidr(ip, net, bits)
            {
                return false;
            }
            if entry.parse::<IpAddr>().ok().is_some_and(|e| Some(e) == ip) {
                return false;
            }
            let (name, entry_port) = split_port(&entry);
            if entry_port.is_some_and(|p| p != port) {
                continue;
            }
            let name = name
                .strip_prefix("*.")
                .map(|n| format!(".{n}"))
                .unwrap_or_else(|| name.to_string());
            let matched = match name.strip_prefix('.') {
                Some(domain) => host.ends_with(&format!(".{domain}")),
                None => host == name || host.ends_with(&format!(".{name}")),
            };
            if matched {
                return false;
            }
        }
        true
    }
}

fn split_port(entry: &str) -> (&str, Option<u16>) {
    if let Some((ip, rest)) = entry.strip_prefix('[').and_then(|e| e.split_once(']')) {
        return (ip, rest.strip_prefix(':').and_then(|p| p.parse().ok()));
    }
    match entry.rsplit_once(':') {
        Some((name, port)) if !name.contains(':') => (name, port.parse().ok()),
        _ => (entry, None),
    }
}

fn in_cidr(ip: IpAddr, net: IpAddr, bits: u32) -> bool {
    match (ip, net) {
        (IpAddr::V4(a), IpAddr::V4(b)) if bits <= 32 => {
            let mask = if bits == 0 {
                0
            } else {
                u32::MAX << (32 - bits)
            };
            u32::from(a) & mask == u32::from(b) & mask
        }
        (IpAddr::V6(a), IpAddr::V6(b)) if bits <= 128 => {
            let mask = if bits == 0 {
                0
            } else {
                u128::MAX << (128 - bits)
            };
            u128::from(a) & mask == u128::from(b) & mask
        }
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::ProxyEnv;

    fn env(pairs: &[(&str, &str)]) -> ProxyEnv {
        let pairs: Vec<(String, String)> = pairs
            .iter()
            .map(|(k, v)| (k.to_string(), v.to_string()))
            .collect();
        ProxyEnv::from_lookup(&move |name| {
            pairs
                .iter()
                .find(|(k, _)| k == name)
                .map(|(_, v)| v.clone())
        })
    }

    // Fault: the Rust CLI ignored the proxy settings the Go CLI honours
    // (its transport reads HTTPS_PROXY, HTTP_PROXY and NO_PROXY), or sent a
    // loopback or NO_PROXY host through the proxy.
    #[test]
    fn proxy_choice_follows_go() {
        let e = env(&[
            ("HTTPS_PROXY", "proxy.example:3128"),
            ("http_proxy", "http://p2:8080"),
            (
                "NO_PROXY",
                "nas01, .lab.example,10.0.0.0/8,192.0.2.10,pve:8443",
            ),
        ]);
        assert_eq!(
            e.for_url("https", "hub.example", 8443).as_deref(),
            Some("http://proxy.example:3128")
        );
        assert_eq!(
            e.for_url("http", "hub.example", 80).as_deref(),
            Some("http://p2:8080")
        );
        for host in [
            "localhost",
            "127.0.0.1",
            "[::1]",
            "nas01",
            "x.nas01",
            "a.lab.example",
            "10.1.2.3",
            "192.0.2.10",
        ] {
            assert_eq!(e.for_url("https", host, 443), None, "{host}");
        }
        assert_eq!(e.for_url("https", "pve", 8443), None);
        assert!(e.for_url("https", "pve", 443).is_some());
        assert!(e.for_url("https", "lab.example", 443).is_some());
        assert_eq!(
            env(&[("NO_PROXY", "*"), ("HTTPS_PROXY", "p")]).for_url("https", "h", 443),
            None
        );
        assert_eq!(env(&[]).for_url("https", "h", 443), None);
    }
}
