//! Go's float text: `strconv.FormatFloat`, `fmt.Print` and `encoding/json`'s rule.
//!
//! Rust's `{}` and `{:e}` print the shortest digits that read back to the
//! same float, as Go's precision -1 does; only the layout differs.

/// Parses JSON number text as Go's `strconv.ParseFloat(s, 64)`: `None` when
/// the value is out of range (Go returns an error there).
pub fn parse_f64(text: &str) -> Option<f64> {
    let value: f64 = text.parse().ok()?;
    value.is_finite().then_some(value)
}

/// `strconv.FormatFloat(f, 'f', -1, 64)`.
pub fn format_f_shortest(f: f64) -> String {
    format!("{f}")
}

/// `strconv.FormatFloat(f, 'f', prec, 64)`.
pub fn format_f(f: f64, prec: usize) -> String {
    format!("{f:.prec$}")
}

/// Splits Rust's `{:e}` text into mantissa and exponent.
fn shortest_exponent(f: f64) -> (String, i32) {
    let text = format!("{f:e}");
    let (mantissa, exp) = text.split_once('e').expect("exponent form");
    (mantissa.to_string(), exp.parse().expect("exponent digits"))
}

/// `strconv.FormatFloat(f, 'e', -1, 64)`: at least two exponent digits.
pub fn format_e_shortest(f: f64) -> String {
    let (mantissa, exp) = shortest_exponent(f);
    let sign = if exp < 0 { '-' } else { '+' };
    format!("{mantissa}e{sign}{:02}", exp.unsigned_abs())
}

/// The float form Go's `encoding/json` writes for a `float64`.
pub fn format_json(f: f64) -> String {
    let abs = f.abs();
    if abs != 0.0 && !(1e-6..1e21).contains(&abs) {
        let mut text = format_e_shortest(f);
        // Go cleans up "e-09" to "e-9".
        let n = text.len();
        let bytes = text.as_bytes();
        if n >= 4 && bytes[n - 4] == b'e' && bytes[n - 3] == b'-' && bytes[n - 2] == b'0' {
            text.remove(n - 2);
        }
        return text;
    }
    format_f_shortest(f)
}

/// `fmt.Sprint(f)` (`%v`): shortest digits, exponent form when the exponent
/// is below -4 or at least 6 (Go 1.27.3 prints 1e+06 for a million).
pub fn format_v(f: f64) -> String {
    if f == 0.0 {
        return format_f_shortest(f);
    }
    let (_, exp) = shortest_exponent(f);
    if !(-4..6).contains(&exp) {
        return format_e_shortest(f);
    }
    format_f_shortest(f)
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: JSON output printed floats in Rust's layout ("1e21", "1e-7")
    // instead of Go's ("1e+21", "1e-7" after Go's clean-up).
    #[test]
    fn json_floats_use_go_layout() {
        assert_eq!(format_json(1e21), "1e+21");
        assert_eq!(format_json(1e20), "100000000000000000000");
        assert_eq!(format_json(1e-7), "1e-7");
        assert_eq!(format_json(1.5e-10), "1.5e-10");
        assert_eq!(format_json(1e-6), "0.000001");
        assert_eq!(format_json(-0.0), "-0");
        assert_eq!(format_json(1e100), "1e+100");
        assert_eq!(format_json(5e-324), "5e-324");
        assert_eq!(format_json(12345678.9), "12345678.9");
        assert_eq!(format_json(1.2345678901234568e29), "1.2345678901234568e+29");
    }

    // Fault: `whoami` printed a numeric scope differently from Go's
    // fmt.Print (probe: 1e+06, 1.23456789e+08, 0.0001, 1e-05, 12.5).
    #[test]
    fn sprint_floats_use_go_layout() {
        assert_eq!(format_v(1e6), "1e+06");
        assert_eq!(format_v(123456789.0), "1.23456789e+08");
        assert_eq!(format_v(0.0001), "0.0001");
        assert_eq!(format_v(0.00001), "1e-05");
        assert_eq!(format_v(12.5), "12.5");
    }
}
