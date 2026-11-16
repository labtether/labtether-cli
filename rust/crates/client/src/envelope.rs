//! The Hub's v2 response envelope, read with serde.

use serde::Deserialize;
use serde_json::Value;

use crate::error::ClientError;

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
    let unreadable = || {
        ClientError::Failed(format!(
            "status {status}: {}",
            String::from_utf8_lossy(body)
        ))
    };
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
