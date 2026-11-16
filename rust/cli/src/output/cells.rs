//! Table cell text, as the Go CLI's `cellText` and its formatters.

use serde_json::Value;

use super::gofloat::{format_f, format_f_shortest, parse_f64};
use super::gojson::{self, encode};

/// A value the Hub left out or sent as null.
pub const EMPTY: &str = "-";

/// How the Go CLI decoded the numbers it prints: with `UseNumber` (table
/// commands keep the Hub's text) or as `float64` (`whoami`).
pub use gojson::Numbers;

/// The Go CLI's `numberText` for a `json.Number`.
pub fn number_text(text: &str) -> String {
    if text.parse::<i64>().is_ok() || text.parse::<u64>().is_ok() {
        return text.to_string();
    }
    let Some(f) = parse_f64(text) else {
        return text.to_string();
    };
    if f == f.trunc() {
        return format_f(f, 0);
    }
    let rounded = format_f(f, 2);
    let trimmed = rounded.trim_end_matches('0').trim_end_matches('.');
    if trimmed == "-0" {
        "0".to_string()
    } else {
        trimmed.to_string()
    }
}

/// The Go CLI's `cellText`.
pub fn cell_text(value: Option<&Value>, numbers: Numbers) -> String {
    match value {
        None | Some(Value::Null) => EMPTY.to_string(),
        Some(Value::String(text)) => text.clone(),
        Some(Value::Bool(true)) => "yes".to_string(),
        Some(Value::Bool(false)) => "no".to_string(),
        Some(Value::Number(number)) => {
            let text = number.to_string();
            match numbers {
                Numbers::Text => number_text(&text),
                // Go formats the float64 with 'f' and -1, then as a json.Number.
                Numbers::Float => {
                    number_text(&parse_f64(&text).map(format_f_shortest).unwrap_or(text))
                }
            }
        }
        Some(Value::Array(items)) => items
            .iter()
            .map(|item| cell_text(Some(item), numbers))
            .filter(|text| text != EMPTY)
            .collect::<Vec<_>>()
            .join(", "),
        Some(object @ Value::Object(_)) => encode(object, numbers, false),
    }
}

pub fn text(value: Option<&Value>) -> String {
    cell_text(value, Numbers::Text)
}

/// The Go CLI's `cleanCell`: one line, no control characters, "-" when
/// empty.
pub fn clean_cell(text: &str) -> String {
    let mapped: String = text
        .chars()
        .map(|c| if c.is_control() { ' ' } else { c })
        .collect();
    let trimmed = mapped.trim();
    if trimmed.is_empty() {
        EMPTY.to_string()
    } else {
        trimmed.to_string()
    }
}

/// The Go CLI's `statusWord`: "stale" is "Not responding", underscores
/// become spaces and the first letter is upper case.
pub fn status_word(value: Option<&Value>) -> String {
    let text = text(value).trim().to_string();
    if matches!(text.to_lowercase().as_str(), "stale" | "unresponsive") {
        return "Not responding".to_string();
    }
    let text = text.replace('_', " ");
    let mut chars = text.chars();
    match chars.next() {
        None => text,
        Some(first) => {
            let mut upper = first.to_uppercase();
            // Go's unicode.ToUpper maps one rune to one rune.
            let first = match (upper.next(), upper.next()) {
                (Some(single), None) => single,
                _ => first,
            };
            std::iter::once(first).chain(chars).collect()
        }
    }
}

/// The Go CLI's `shortID`: a 64-character Docker ID cut to 12 characters.
pub fn short_id(value: Option<&Value>) -> String {
    let text = text(value);
    if text.len() > 12 && text.to_lowercase().bytes().all(|b| b.is_ascii_hexdigit()) {
        return text[..12].to_string();
    }
    text
}

fn number_value(value: Option<&Value>) -> Option<f64> {
    match value {
        Some(Value::Number(number)) => parse_f64(&number.to_string()),
        _ => None,
    }
}

/// The Go CLI's `byteSize`: binary units such as "1.5 GiB".
pub fn byte_size(value: Option<&Value>) -> String {
    let Some(mut size) = number_value(value) else {
        return text(value);
    };
    const UNITS: [&str; 6] = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
    let mut unit = 0;
    while size >= 1024.0 && unit < UNITS.len() - 1 {
        size /= 1024.0;
        unit += 1;
    }
    if unit == 0 {
        return format!("{} B", format_f(size, 0));
    }
    format!("{} {}", format_f(size, 1), UNITS[unit])
}

/// The Go CLI's `unixTimeText`: Unix seconds as local minutes.
pub fn unix_time_text(value: Option<&Value>) -> String {
    match number_value(value) {
        Some(seconds) if seconds > 0.0 => super::time::format_unix_minutes(seconds as i64),
        _ => text(value),
    }
}

/// The Go CLI's `timeText`: an RFC 3339 time as local minutes.
pub fn time_text(value: Option<&Value>) -> String {
    let text = text(value);
    match super::time::parse_rfc3339(&text) {
        None => text,
        Some(t) if t.is_zero() => EMPTY.to_string(),
        Some(t) => super::time::format_local(&t),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn num(text: &str) -> Value {
        serde_json::from_str(text).unwrap()
    }

    // Fault: a large integer printed as a float, a fraction printed with
    // many digits, or a half rounded the other way from Go's FormatFloat.
    #[test]
    fn numbers_print_like_go() {
        assert_eq!(number_text("12345678901234567890"), "12345678901234567890");
        assert_eq!(number_text("-0"), "-0");
        assert_eq!(number_text("2.50"), "2.5");
        assert_eq!(number_text("0.125"), "0.12");
        assert_eq!(number_text("0.375"), "0.38");
        assert_eq!(number_text("-0.001"), "0");
        assert_eq!(number_text("1e3"), "1000");
        assert_eq!(number_text("1e400"), "1e400");
        assert_eq!(
            cell_text(Some(&num("12345678901234567890")), Numbers::Float),
            "12345678901234567000"
        );
        assert_eq!(byte_size(Some(&num("1536"))), "1.5 KiB");
        assert_eq!(byte_size(Some(&num("2.5"))), "2 B");
        assert_eq!(byte_size(Some(&Value::String("n/a".into()))), "n/a");
    }

    // Fault: a remote host could put terminal control characters or line
    // breaks in a name and break or spoof the table.
    #[test]
    fn cells_drop_control_characters() {
        assert_eq!(clean_cell("web\u{1b}[31m\nnode\t1"), "web [31m node 1");
        assert_eq!(clean_cell("  \u{85} "), "-");
        assert_eq!(
            status_word(Some(&Value::String("stale".into()))),
            "Not responding"
        );
        assert_eq!(
            status_word(Some(&Value::String("in_progress".into()))),
            "In progress"
        );
    }
}
