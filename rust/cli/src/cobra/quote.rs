//! Go's `strconv.Quote` and `strconv.QuoteRune`, used by `%q` in the error
//! messages the Go CLI prints.

/// Unicode format characters (category Cf) and the non-ASCII spaces, which
/// Go's `unicode.IsPrint` rejects. Unassigned code points are treated as
/// printable here; Go escapes them.
fn is_print(c: char) -> bool {
    let code = c as u32;
    if code < 0x80 {
        return (0x20..0x7f).contains(&code);
    }
    if c.is_control() {
        return false;
    }
    const NOT_PRINT: &[(u32, u32)] = &[
        (0x00a0, 0x00a0),
        (0x00ad, 0x00ad),
        (0x0600, 0x0605),
        (0x061c, 0x061c),
        (0x06dd, 0x06dd),
        (0x070f, 0x070f),
        (0x0890, 0x0891),
        (0x08e2, 0x08e2),
        (0x1680, 0x1680),
        (0x180e, 0x180e),
        (0x2000, 0x200f),
        (0x2028, 0x202f),
        (0x205f, 0x2064),
        (0x2066, 0x206f),
        (0x3000, 0x3000),
        (0xd800, 0xf8ff),
        (0xfeff, 0xfeff),
        (0xfff9, 0xfffb),
        (0x110bd, 0x110bd),
        (0x110cd, 0x110cd),
        (0x13430, 0x1343f),
        (0x1bca0, 0x1bca3),
        (0x1d173, 0x1d17a),
        (0xe0001, 0xe0001),
        (0xe0020, 0xe007f),
        (0xf0000, 0x10ffff),
    ];
    !NOT_PRINT
        .iter()
        .any(|&(low, high)| (low..=high).contains(&code))
}

fn push_escaped(out: &mut String, c: char, quote: char) {
    if c == quote || c == '\\' {
        out.push('\\');
        out.push(c);
        return;
    }
    if is_print(c) {
        out.push(c);
        return;
    }
    match c {
        '\u{7}' => out.push_str("\\a"),
        '\u{8}' => out.push_str("\\b"),
        '\u{c}' => out.push_str("\\f"),
        '\n' => out.push_str("\\n"),
        '\r' => out.push_str("\\r"),
        '\t' => out.push_str("\\t"),
        '\u{b}' => out.push_str("\\v"),
        _ => {
            let code = c as u32;
            if code < 0x20 || code == 0x7f {
                out.push_str(&format!("\\x{code:02x}"));
            } else if code < 0x10000 {
                out.push_str(&format!("\\u{code:04x}"));
            } else {
                out.push_str(&format!("\\U{code:08x}"));
            }
        }
    }
}

/// `strconv.Quote(text)`.
pub fn quote(text: &str) -> String {
    let mut out = String::with_capacity(text.len() + 2);
    out.push('"');
    for c in text.chars() {
        push_escaped(&mut out, c, '"');
    }
    out.push('"');
    out
}

/// `strconv.QuoteRune(c)`, as `%q` prints a rune.
pub fn quote_rune(c: char) -> String {
    let mut out = String::from("'");
    push_escaped(&mut out, c, '\'');
    out.push('\'');
    out
}

#[cfg(test)]
mod tests {
    use super::{quote, quote_rune};

    // Fault: an argument with a control or invisible character printed raw
    // in an error message, where the Go CLI escapes it.
    #[test]
    fn quote_escapes_like_go() {
        assert_eq!(
            quote("a\u{0}b\u{200b}c\u{a0}d\te\"f\\g\u{e9}"),
            "\"a\\x00b\\u200bc\\u00a0d\\te\\\"f\\\\g\u{e9}\""
        );
        assert_eq!(quote("x y"), "\"x y\"");
        assert_eq!(quote_rune('x'), "'x'");
        assert_eq!(quote_rune('\''), "'\\''");
    }
}
