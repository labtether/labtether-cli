//! JSON text as the Go CLI prints it. The Go CLI decodes the Hub's answer
//! into `any` and encodes it again, so keys come out sorted (serde_json's
//! default map is sorted the same way), numbers in Go's float64 layout and
//! strings with Go's escapes (`<`, `>`, `&`, U+2028 and U+2029 as `\uXXXX`).
//! This is a `serde_json` formatter over `serde_json::Value`.

use std::io::{self, Write};

use serde::Serialize;
use serde_json::ser::{CharEscape, Formatter};
use serde_json::{Serializer, Value};

use crate::gofloat::{format_json, parse_f64};

/// How numbers are written.
#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Numbers {
    /// As Go prints a `float64` (the `any` round trip). A number too large
    /// for a float64 keeps its text.
    Float,
    /// The Hub's own text, as Go prints a `json.Number`.
    Text,
}

/// A Go-style formatter: indented with two spaces, or compact.
struct GoFormatter {
    numbers: Numbers,
    indent: bool,
    depth: usize,
    has_value: bool,
}

impl GoFormatter {
    fn newline<W: ?Sized + Write>(&self, w: &mut W) -> io::Result<()> {
        if self.indent {
            w.write_all(b"\n")?;
            for _ in 0..self.depth {
                w.write_all(b"  ")?;
            }
        }
        Ok(())
    }

    fn begin<W: ?Sized + Write>(&mut self, w: &mut W, open: &[u8]) -> io::Result<()> {
        self.depth += 1;
        self.has_value = false;
        w.write_all(open)
    }

    fn end<W: ?Sized + Write>(&mut self, w: &mut W, close: &[u8]) -> io::Result<()> {
        self.depth -= 1;
        if self.has_value {
            self.newline(w)?;
        }
        w.write_all(close)
    }

    fn item<W: ?Sized + Write>(&mut self, w: &mut W, first: bool) -> io::Result<()> {
        if !first {
            w.write_all(b",")?;
        }
        self.newline(w)
    }
}

impl Formatter for GoFormatter {
    fn write_number_str<W: ?Sized + Write>(&mut self, w: &mut W, value: &str) -> io::Result<()> {
        match (self.numbers, parse_f64(value)) {
            (Numbers::Float, Some(f)) => w.write_all(format_json(f).as_bytes()),
            _ => w.write_all(value.as_bytes()),
        }
    }

    fn write_string_fragment<W: ?Sized + Write>(
        &mut self,
        w: &mut W,
        fragment: &str,
    ) -> io::Result<()> {
        let mut start = 0;
        for (i, c) in fragment.char_indices() {
            if matches!(c, '<' | '>' | '&' | '\u{2028}' | '\u{2029}') {
                w.write_all(&fragment.as_bytes()[start..i])?;
                write!(w, "\\u{:04x}", c as u32)?;
                start = i + c.len_utf8();
            }
        }
        w.write_all(&fragment.as_bytes()[start..])
    }

    fn write_char_escape<W: ?Sized + Write>(
        &mut self,
        w: &mut W,
        escape: CharEscape,
    ) -> io::Result<()> {
        let text: &[u8] = match escape {
            CharEscape::Quote => b"\\\"",
            CharEscape::ReverseSolidus => b"\\\\",
            CharEscape::Solidus => b"/",
            CharEscape::Backspace => b"\\b",
            CharEscape::FormFeed => b"\\f",
            CharEscape::LineFeed => b"\\n",
            CharEscape::CarriageReturn => b"\\r",
            CharEscape::Tab => b"\\t",
            CharEscape::AsciiControl(byte) => return write!(w, "\\u{:04x}", byte),
        };
        w.write_all(text)
    }

    fn begin_array<W: ?Sized + Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.begin(w, b"[")
    }

    fn end_array<W: ?Sized + Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.end(w, b"]")
    }

    fn begin_array_value<W: ?Sized + Write>(&mut self, w: &mut W, first: bool) -> io::Result<()> {
        self.item(w, first)
    }

    fn end_array_value<W: ?Sized + Write>(&mut self, _w: &mut W) -> io::Result<()> {
        self.has_value = true;
        Ok(())
    }

    fn begin_object<W: ?Sized + Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.begin(w, b"{")
    }

    fn end_object<W: ?Sized + Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.end(w, b"}")
    }

    fn begin_object_key<W: ?Sized + Write>(&mut self, w: &mut W, first: bool) -> io::Result<()> {
        self.item(w, first)
    }

    fn begin_object_value<W: ?Sized + Write>(&mut self, w: &mut W) -> io::Result<()> {
        w.write_all(if self.indent { b": " } else { b":" })
    }

    fn end_object_value<W: ?Sized + Write>(&mut self, _w: &mut W) -> io::Result<()> {
        self.has_value = true;
        Ok(())
    }
}

/// Encodes `value` the Go way: `indent` for `json.Encoder.SetIndent("",
/// "  ")`, compact for `json.Marshal`.
pub fn encode(value: &Value, numbers: Numbers, indent: bool) -> String {
    let mut out = Vec::new();
    let formatter = GoFormatter {
        numbers,
        indent,
        depth: 0,
        has_value: false,
    };
    let mut serializer = Serializer::with_formatter(&mut out, formatter);
    value
        .serialize(&mut serializer)
        .expect("writing to memory cannot fail");
    String::from_utf8(out).expect("serde_json writes UTF-8")
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: JSON output changed key order, escaping, number form or
    // indentation from what the Go CLI prints (checked against Go 1.27.3,
    // evidence/logs/probe1.log), which breaks scripts that compare output.
    #[test]
    fn output_matches_go_encoder() {
        let value: Value = serde_json::from_str(
            r#"{"name":"<node>","nested":{"b":true,"a":[1,null]},"x":1.50,"e":{},"l":[],"big":1e21,"tiny":1e-7,"s":"a\b\f\u0001\u007f &"}"#,
        )
        .unwrap();
        let want = "{\n  \"big\": 1e+21,\n  \"e\": {},\n  \"l\": [],\n  \"name\": \"\\u003cnode\\u003e\",\n  \"nested\": {\n    \"a\": [\n      1,\n      null\n    ],\n    \"b\": true\n  },\n  \"s\": \"a\\b\\f\\u0001\u{7f}\\u2028\\u0026\",\n  \"tiny\": 1e-7,\n  \"x\": 1.5\n}";
        assert_eq!(encode(&value, Numbers::Float, true), want);
        // Text mode keeps a number's digits; serde_json writes an exponent
        // with its sign ("1e+21") where Go keeps the Hub's text ("1e21").
        let compact = encode(&value, Numbers::Text, false);
        let want = r#"{"big":1e+21,"e":{},"l":[],"name":"\u003cnode\u003e","#;
        assert!(compact.starts_with(want), "{compact}");
        assert!(compact.ends_with(r#""x":1.50}"#), "{compact}");
    }
}
