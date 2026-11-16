//! Go's `url.PathEscape` and `url.QueryEscape`, so request paths and
//! queries are byte for byte what the Go CLI sends.

#[derive(Clone, Copy, PartialEq, Eq)]
enum Mode {
    PathSegment,
    QueryComponent,
}

fn should_escape(c: u8, mode: Mode) -> bool {
    if c.is_ascii_alphanumeric() {
        return false;
    }
    match c {
        b'-' | b'_' | b'.' | b'~' => return false,
        b'$' | b'&' | b'+' | b',' | b'/' | b':' | b';' | b'=' | b'?' | b'@' => {
            return match mode {
                Mode::PathSegment => matches!(c, b'/' | b';' | b',' | b'?'),
                Mode::QueryComponent => true,
            };
        }
        _ => {}
    }
    true
}

fn escape(text: &str, mode: Mode) -> String {
    let mut out = String::with_capacity(text.len());
    for &c in text.as_bytes() {
        if c == b' ' && mode == Mode::QueryComponent {
            out.push('+');
        } else if should_escape(c, mode) {
            out.push_str(&format!("%{c:02X}"));
        } else {
            out.push(char::from(c));
        }
    }
    out
}

/// Go's `url.PathEscape`.
pub fn path_escape(text: &str) -> String {
    escape(text, Mode::PathSegment)
}

/// Go's `url.QueryEscape`.
pub fn query_escape(text: &str) -> String {
    escape(text, Mode::QueryComponent)
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: an ID with "/" or "?" changed the request path or query, so a
    // command read a different Hub object than the one named.
    #[test]
    fn escapes_match_go() {
        assert_eq!(path_escape("ct/one?x=1"), "ct%2Fone%3Fx=1");
        assert_eq!(path_escape("a b;c,d"), "a%20b%3Bc%2Cd");
        assert_eq!(query_escape("/var/log x&y"), "%2Fvar%2Flog+x%26y");
    }
}
