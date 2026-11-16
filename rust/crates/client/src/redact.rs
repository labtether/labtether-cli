//! Hiding secret values in Hub answers before the CLI prints them, as the
//! Go CLI's `internal/redact` does: in `--json` output, table cells and the
//! error text for an answer the CLI cannot read. One set of key rules for
//! all three.

use serde_json::{Map, Value};

use crate::gofloat::parse_f64;
use crate::gojson::{Numbers, encode};

/// What a secret value is replaced with.
pub const HIDDEN: &str = "[redacted]";

const SECRET_WORDS: [&str; 5] = ["password", "secret", "token", "api_key", "private_key"];

/// Go's `strings.ToLower`: one lower-case rune per rune (Unicode simple case
/// mapping). Rust's `to_lowercase` turns `İ` (U+0130) into two characters,
/// so `PRİVATE_KEY` would not read as `private_key` there.
fn go_lower(text: &str) -> String {
    text.chars()
        .map(|c| {
            if c == '\u{130}' {
                return 'i';
            }
            let mut lower = c.to_lowercase();
            match (lower.next(), lower.next()) {
                (Some(single), None) => single,
                _ => c,
            }
        })
        .collect()
}

/// Go's `redact.MentionsSecret`: the text holds a word that names a secret
/// field.
pub fn mentions_secret(text: &str) -> bool {
    let text = go_lower(text);
    SECRET_WORDS.iter().any(|word| text.contains(word))
}

/// Go's `redact.SensitiveKey`.
pub fn is_sensitive_key(key: &str) -> bool {
    let normalized = go_lower(&key.replace(['-', ' '], "_"));
    mentions_secret(&normalized) || normalized == "apikey"
}

/// Go's `safeSecretStatus`: values that say whether a secret is set without
/// showing it.
fn is_safe_status(value: &Value) -> bool {
    match value {
        Value::String(text) => matches!(text.as_str(), "(set)" | "(not set)" | HIDDEN),
        Value::Bool(_) => true,
        _ => false,
    }
}

/// Go's `redact.Value`: a copy with every secret value hidden, at every
/// depth, and how many were hidden.
pub fn redact_counted(value: &Value) -> (Value, usize) {
    let mut hidden = 0;
    let copy = walk(value, &mut hidden);
    (copy, hidden)
}

/// A copy of `value` with every secret value hidden.
pub fn redact(value: &Value) -> Value {
    redact_counted(value).0
}

fn walk(value: &Value, hidden: &mut usize) -> Value {
    match value {
        Value::Object(members) => Value::Object(
            members
                .iter()
                .map(|(key, child)| {
                    let replaced = if is_sensitive_key(key) && !is_safe_status(child) {
                        *hidden += 1;
                        Value::String(HIDDEN.into())
                    } else {
                        walk(child, hidden)
                    };
                    (key.clone(), replaced)
                })
                .collect::<Map<String, Value>>(),
        ),
        Value::Array(items) => Value::Array(items.iter().map(|item| walk(item, hidden)).collect()),
        other => other.clone(),
    }
}

/// How Go's `redact.Decode` holds the numbers of a decoded answer: as
/// float64, unless one is beyond float64; then every number keeps its text.
pub fn decoded_numbers(value: &Value) -> Numbers {
    fn beyond(value: &Value) -> bool {
        match value {
            Value::Number(number) => parse_f64(&number.to_string()).is_none(),
            Value::Array(items) => items.iter().any(beyond),
            Value::Object(members) => members.values().any(beyond),
            _ => false,
        }
    }
    if beyond(value) {
        Numbers::Text
    } else {
        Numbers::Float
    }
}

/// Go's `unreadableAnswerText`: an answer that is not a v2 envelope, as
/// error text. Text that is not JSON is shown as it came unless it names a
/// secret field; JSON is shown as it came when it holds no secret value,
/// and otherwise re-encoded with its secrets hidden.
pub fn unreadable_answer_text(body: &[u8]) -> String {
    let Ok(decoded) = serde_json::from_slice::<Value>(body) else {
        let text = String::from_utf8_lossy(body);
        if mentions_secret(&text) {
            return format!(
                "the answer is not valid JSON and names a secret field, so its {} bytes are not shown",
                body.len()
            );
        }
        return text.into_owned();
    };
    let (redacted, hidden) = redact_counted(&decoded);
    if hidden == 0 {
        return String::from_utf8_lossy(body).into_owned();
    }
    encode(&redacted, decoded_numbers(&decoded), false)
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: a key spelt with `İ` (U+0130), such as `PRİVATE_KEY`, was not
    // seen as secret, because Rust lower-cases `İ` to two characters where Go
    // gives `i`. Its value then printed in clear.
    #[test]
    fn keys_lower_case_as_go() {
        assert!(is_sensitive_key("PR\u{130}VATE_KEY"));
        assert!(is_sensitive_key("TO\u{212A}EN"));
        assert!(is_sensitive_key("Client Secret"));
        assert!(!is_sensitive_key("name"));
    }
}
