//! The Hub's v2 response envelope, read with serde.

use serde::Deserialize;
use serde_json::Value;

use crate::error::ClientError;
use crate::redact::unreadable_answer_text;

/// The envelope fields the Go CLI's `V2Response` declares. A field with the
/// wrong type makes the answer unreadable, as in Go.
#[derive(Deserialize)]
#[allow(dead_code)]
struct Envelope {
    request_id: Option<String>,
    error: Option<String>,
    message: Option<String>,
    status: Option<i64>,
    meta: Option<Meta>,
}

#[derive(Deserialize)]
#[allow(dead_code)]
struct Meta {
    total: Option<i64>,
    page: Option<i64>,
    per_page: Option<i64>,
}

/// Decodes a Hub answer into its `data` payload, or an error: "status
/// <code>: <body>" for a body that is not a readable envelope, and
/// "<message> (status <code>)" for an error status, as the Go CLI says.
pub fn decode(status: u16, body: &[u8]) -> Result<Value, ClientError> {
    // Not an envelope: the answer is shown with its secrets hidden.
    let unreadable =
        || ClientError::Failed(format!("status {status}: {}", unreadable_answer_text(body)));
    let top: Value = serde_json::from_slice(body).map_err(|_| unreadable())?;
    let envelope = match &top {
        Value::Null => None,
        Value::Object(_) => Some(Envelope::deserialize(&top).map_err(|_| unreadable())?),
        _ => return Err(unreadable()),
    };
    if status >= 400 {
        let envelope = envelope.unwrap_or(Envelope {
            request_id: None,
            error: None,
            message: None,
            status: None,
            meta: None,
        });
        let message = envelope
            .message
            .filter(|m| !m.is_empty())
            .or(envelope.error)
            .unwrap_or_default();
        return Err(ClientError::Status { status, message });
    }
    // A few v2 routes answer with the payload itself instead of a data
    // envelope; the Go CLI uses the whole body then.
    Ok(match top.get("data") {
        Some(data) => data.clone(),
        None => top,
    })
}

/// The Go CLI's `Download` error for a status outside 200-299: the Hub's
/// message (or error) when the body is a readable envelope that has one,
/// else "request failed". Either way " (status N)" follows.
pub fn download_error(status: u16, body: &[u8]) -> ClientError {
    let message = serde_json::from_slice::<Value>(body)
        .ok()
        .and_then(|top| match &top {
            Value::Object(_) => Envelope::deserialize(&top).ok(),
            _ => None,
        })
        .and_then(|envelope| {
            envelope
                .message
                .filter(|m| !m.is_empty())
                .or(envelope.error)
        })
        .filter(|m| !m.is_empty());
    ClientError::Status {
        status,
        message: message.unwrap_or_else(|| "request failed".to_string()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: a 404 or 401 lost its Hub message or status, so the CLI printed
    // a different error and could not pick exit code 3 or 2.
    #[test]
    fn error_status_keeps_message_and_code() {
        let err = decode(404, br#"{"error":"not_found","message":"asset not found"}"#).unwrap_err();
        assert_eq!(
            err,
            ClientError::Status {
                status: 404,
                message: "asset not found".into()
            }
        );
        assert_eq!(err.to_string(), "asset not found (status 404)");
        assert_eq!(
            decode(401, br#"{"error":"unauthorized"}"#)
                .unwrap_err()
                .to_string(),
            "unauthorized (status 401)"
        );
        assert_eq!(decode(500, b"{}").unwrap_err().to_string(), " (status 500)");
    }

    // Fault: a body the Go CLI cannot read as its envelope (HTML, or a field
    // of the wrong type) was accepted, hiding a Hub or proxy error.
    #[test]
    fn unreadable_bodies_report_status_and_body() {
        assert_eq!(
            decode(502, b"<html>bad gateway</html>")
                .unwrap_err()
                .to_string(),
            "status 502: <html>bad gateway</html>"
        );
        assert_eq!(
            decode(200, br#"{"status":"x","data":1}"#)
                .unwrap_err()
                .to_string(),
            r#"status 200: {"status":"x","data":1}"#
        );
        assert!(decode(200, b"[1]").is_err());
    }

    fn text(body: &[u8]) -> String {
        decode(200, body).unwrap_err().to_string()["status 200: ".len()..].to_string()
    }

    // Fault: an answer the CLI could not read as its envelope (a proxy page,
    // or a field of the wrong type) was echoed raw in the error text, secrets
    // included. labtether-cli #49 fixed this in Go; the words and bytes here
    // are Go's.
    #[test]
    fn unreadable_answers_hide_secrets() {
        assert_eq!(
            text(b"<html>bad gateway</html>"),
            "<html>bad gateway</html>"
        );
        assert_eq!(
            text(b"<p>Token: lt_abc</p>"),
            "the answer is not valid JSON and names a secret field, so its 20 bytes are not shown"
        );
        assert_eq!(
            text(br#"{"status":"x", "data":1}"#),
            r#"{"status":"x", "data":1}"#
        );
        assert_eq!(
            text(br#"{"status":"x","data":{"api_key":"lt_abc","n":1.50,"h":"<b>"}}"#),
            r#"{"data":{"api_key":"[redacted]","h":"\u003cb\u003e","n":1.5},"status":"x"}"#
        );
        // A number beyond float64: every number keeps its text, as Go
        // (serde_json spells the exponent "1e+400" where Go keeps "1e400").
        assert_eq!(
            text(br#"[{"password":"p","n":1e400,"m":2.50}]"#),
            r#"[{"m":2.50,"n":1e+400,"password":"[redacted]"}]"#
        );
    }

    // Fault: `"data": null` must stay null, and a missing data member gives
    // the whole body, as in the Go CLI.
    #[test]
    fn data_selection_follows_go() {
        assert_eq!(decode(200, br#"{"data":null}"#).unwrap(), Value::Null);
        assert!(
            decode(200, br#"{"devices":[]}"#)
                .unwrap()
                .get("devices")
                .is_some()
        );
    }
}
