//! One Hub request with the Go client's redirect rules: same-origin only,
//! at most 10, and Go's method and body changes on each kind of redirect.

use std::time::Instant;

use ureq::Body;
use ureq::http::Response;
use url::Url;

use crate::error::ClientError;
use crate::{Client, MAX_REDIRECTS, same_origin, shown, transport, validate_base_url};

/// A request method the CLI sends.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Method {
    Get,
    Post,
    Patch,
}

impl Method {
    fn as_str(self) -> &'static str {
        match self {
            Method::Get => "GET",
            Method::Post => "POST",
            Method::Patch => "PATCH",
        }
    }

    /// Go's `urlErrorOp`: the method as it starts an error text.
    fn op(self) -> &'static str {
        match self {
            Method::Get => "Get",
            Method::Post => "Post",
            Method::Patch => "Patch",
        }
    }
}

/// What one hop sends. Go sends `Content-Length: 0` for a POST or PATCH
/// with no body, and `Content-Type` only with a body.
struct Hop<'a> {
    method: Method,
    body: Option<&'a [u8]>,
    content_type: bool,
}

impl Client {
    /// Sends `method path` with an optional JSON `body` and follows
    /// same-origin redirects: the final answer, its body not read yet.
    ///
    /// Go's rules: 301, 302 and 303 turn a POST or PATCH into a GET and drop
    /// the body and its `Content-Type`, for this hop and every later one;
    /// 307 and 308 send the same method and body again.
    pub(crate) fn open(
        &self,
        method: Method,
        path: &str,
        body: Option<&[u8]>,
    ) -> Result<Response<Body>, ClientError> {
        validate_base_url(&self.base_url).map_err(ClientError::Failed)?;
        let origin = Url::parse(&self.base_url)
            .map_err(|_| ClientError::Failed("invalid hub URL".into()))?;
        let mut current = Url::parse(&format!("{}{path}", self.base_url)).map_err(|_| {
            ClientError::Failed(format!("create request: invalid URL {:?}", shown(&origin)))
        })?;
        let op = method.op();
        let mut hop = Hop {
            method,
            body,
            content_type: body.is_some(),
        };
        let deadline = Instant::now() + self.timeout;
        let mut previous: Option<Url> = None;
        for sent in 0..=MAX_REDIRECTS + 1 {
            let failed = |message: String| {
                ClientError::Failed(format!(
                    "request failed: {op} \"{}\": {message}",
                    shown(&current)
                ))
            };
            let left = deadline.saturating_duration_since(Instant::now());
            if left.is_zero() {
                return Err(failed("timed out".into()));
            }
            let agent =
                transport::agent(&self.roots, &self.proxy, &current, left).map_err(failed)?;
            let mut builder = ureq::http::Request::builder()
                .method(hop.method.as_str())
                .uri(current.as_str())
                .header("Authorization", format!("Bearer {}", self.api_key.expose()));
            if hop.content_type {
                builder = builder.header("Content-Type", "application/json");
            }
            if let Some(referer) = previous
                .as_ref()
                .filter(|last| !(last.scheme() == "https" && current.scheme() == "http"))
            {
                builder = builder.header("Referer", shown(referer));
            }
            let response = match (hop.method, hop.body) {
                (Method::Get, _) => builder.body(()).map(|r| agent.run(r)),
                (_, Some(bytes)) => builder.body(bytes).map(|r| agent.run(r)),
                (_, None) => builder.body(&[][..]).map(|r| agent.run(r)),
            }
            .map_err(|_| failed("the request could not be built".into()))?
            .map_err(|e| failed(transport::plain(&e)))?;
            let status = response.status().as_u16();
            let location = response
                .headers()
                .get("location")
                .and_then(|v| v.to_str().ok())
                .map(str::to_string);
            let Some(location) =
                location.filter(|l| !l.is_empty() && matches!(status, 301 | 302 | 303 | 307 | 308))
            else {
                return Ok(response);
            };
            let refused = |message: String| {
                ClientError::Failed(format!("request failed: {op} \"{location}\": {message}"))
            };
            let next = current.join(&location).map_err(|_| {
                failed(format!(
                    "the hub sent an unreadable redirect location {location:?}"
                ))
            })?;
            // Go's client checks before following: with 10 requests sent it
            // refuses the 10th redirect.
            if sent + 1 >= MAX_REDIRECTS {
                return Err(refused(format!("stopped after {MAX_REDIRECTS} redirects")));
            }
            if !same_origin(&origin, &next) {
                return Err(refused(format!(
                    "refusing cross-origin redirect to {}",
                    shown(&next)
                )));
            }
            if matches!(status, 301..=303) {
                if hop.method != Method::Get {
                    hop.method = Method::Get;
                }
                hop.body = None;
                hop.content_type = false;
            }
            previous = Some(std::mem::replace(&mut current, next));
        }
        unreachable!("the loop returns within MAX_REDIRECTS + 1 hops")
    }
}
