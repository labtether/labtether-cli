//! Output: JSON printing with the Go CLI's redaction, tables and cells.

pub mod cells;
pub mod table;
pub mod time;

use labtether_cli_client::redact::decoded_numbers;
pub use labtether_cli_client::redact::{is_sensitive_key, redact};
pub use labtether_cli_client::{gofloat, gojson};
use serde_json::Value;

use gojson::encode;
/// Standard output and error for one run, written when the run ends.
#[derive(Default)]
pub struct Out {
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    /// Set once a JSON result is printed, so a later error does not add a
    /// second JSON document (the Go CLI's `jsonResponseWritten`).
    pub json_written: bool,
    /// Where `files cat` streams a file as it arrives (the process's
    /// standard output). Without one it is kept in `stdout`.
    pub sink: Option<Box<dyn std::io::Write>>,
}

impl Out {
    /// A writer for output that streams: what is buffered so far goes
    /// first, so the order stays as printed.
    pub fn raw_writer(&mut self) -> &mut dyn std::io::Write {
        match &mut self.sink {
            Some(sink) => {
                let _ = sink.write_all(&self.stdout);
                self.stdout.clear();
                sink.as_mut()
            }
            None => &mut self.stdout,
        }
    }

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
/// through Go's `any` (`redact.Decode`), secrets redacted, indented, with a
/// newline. With a number beyond float64 every number keeps its text, as Go.
pub fn json_text(value: &Value) -> String {
    let mut text = encode(&redact(value), decoded_numbers(value), true);
    text.push('\n');
    text
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
            r#"{"host":"nas01","api_key":"lt_secret","api_key_status":"(set)","nested":{"access_token":"t1","name":"visible","items":[{"Private-Key":"k"}],"token_configured":true},"client secret":"s","n":1e400,"m":2.50}"#,
        )
        .unwrap();
        let text = json_text(&value);
        for leaked in ["lt_secret", "t1", "\"k\"", "\"s\""] {
            assert!(!text.contains(leaked), "{leaked} leaked: {text}");
        }
        assert!(text.contains("\"api_key_status\": \"(set)\""));
        assert!(text.contains("\"token_configured\": true"));
        assert!(text.contains("visible"));
        // A number beyond float64: still redacted, and every number keeps
        // its text, as Go since #49 (Go printed this answer raw before).
        assert!(text.contains("\"m\": 2.50"), "{text}");
        assert!(
            text.contains("\"n\": 1e+400") || text.contains("\"n\": 1e400"),
            "{text}"
        );
    }
}
