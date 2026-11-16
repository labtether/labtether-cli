//! Output: JSON printing with the Go CLI's redaction, tables and cells.

pub mod cells;
pub mod gofloat;
pub mod gojson;
pub mod table;
pub mod time;

use serde_json::{Map, Value};

use gojson::{Numbers, encode};
/// Standard output and error for one run, written when the run ends.
#[derive(Default)]
pub struct Out {
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    /// Set once a JSON result is printed, so a later error does not add a
    /// second JSON document (the Go CLI's `jsonResponseWritten`).
    pub json_written: bool,
}

impl Out {
    pub fn print(&mut self, text: &str) {
        self.stdout.extend_from_slice(text.as_bytes());
    }

    pub fn eprint(&mut self, text: &str) {
        self.stderr.extend_from_slice(text.as_bytes());
    }

    /// The Go CLI's `printJSON`.
    pub fn print_json(&mut self, value: &Value) {
        let text = json_text(value);
        self.print(&text);
        self.json_written = true;
    }
}

/// What the Go CLI's `writeJSON` prints: the value after a round trip
/// through Go's `any`, secrets redacted, indented, with a newline. Go
/// prints a payload it cannot decode (a number beyond float64) raw and
/// unredacted; this always redacts and keeps such a number's text.
pub fn json_text(value: &Value) -> String {
    let mut text = encode(&redact(value), Numbers::Float, true);
    text.push('\n');
    text
}

/// The Go CLI's `isSensitiveJSONKey`.
pub fn is_sensitive_key(key: &str) -> bool {
    let normalized = key.replace(['-', ' '], "_").to_lowercase();
    ["password", "secret", "token", "api_key", "private_key"]
        .iter()
        .any(|w| normalized.contains(w))
        || normalized == "apikey"
}

/// The Go CLI's `isSafeSecretStatus`: values that say whether a secret is
/// set without showing it.
fn is_safe_status(value: &Value) -> bool {
    match value {
        Value::String(text) => matches!(text.as_str(), "(set)" | "(not set)" | "[redacted]"),
        Value::Bool(_) => true,
        _ => false,
    }
}

/// The Go CLI's `redactDecodedJSON`.
pub fn redact(value: &Value) -> Value {
    match value {
        Value::Object(members) => Value::Object(
            members
                .iter()
                .map(|(key, child)| {
                    let replaced = if is_sensitive_key(key) && !is_safe_status(child) {
                        Value::String("[redacted]".into())
                    } else {
                        redact(child)
                    };
                    (key.clone(), replaced)
                })
                .collect::<Map<String, Value>>(),
        ),
        Value::Array(items) => Value::Array(items.iter().map(redact).collect()),
        other => other.clone(),
    }
}

/// A JSON object from string pairs, for the CLI's own small results.
pub fn object(pairs: &[(&str, Value)]) -> Value {
    Value::Object(
        pairs
            .iter()
            .map(|(k, v)| (k.to_string(), v.clone()))
            .collect(),
    )
}

pub fn string(text: &str) -> Value {
    Value::String(text.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: a token, password or key in a Hub payload was printed with
    // --json. The Go CLI replaces those values with "[redacted]" and keeps
    // "(set)"/"(not set)" and booleans so status fields stay readable.
    #[test]
    fn secrets_are_redacted_at_every_depth() {
        let value: Value = serde_json::from_str(
            r#"{"host":"nas01","api_key":"lt_secret","api_key_status":"(set)","nested":{"access_token":"t1","name":"visible","items":[{"Private-Key":"k"}],"token_configured":true},"client secret":"s","n":1e400}"#,
        )
        .unwrap();
        let text = json_text(&value);
        for leaked in ["lt_secret", "t1", "\"k\"", "\"s\""] {
            assert!(!text.contains(leaked), "{leaked} leaked: {text}");
        }
        assert!(text.contains("\"api_key_status\": \"(set)\""));
        assert!(text.contains("\"token_configured\": true"));
        assert!(text.contains("visible"));
        // Go prints this payload raw and unredacted (a number beyond
        // float64); the Rust CLI still redacts it.
        assert!(
            text.contains("\"n\": 1e+400") || text.contains("\"n\": 1e400"),
            "{text}"
        );
    }
}
