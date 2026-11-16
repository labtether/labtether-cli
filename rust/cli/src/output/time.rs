//! Times in table cells, with Go's RFC 3339 parser and Go's choice of local
//! time zone. Unix: `TZ` unset gives /etc/localtime, `TZ=""` UTC, a name or
//! path that zone, anything unreadable UTC. Windows: the system zone (jiff
//! reads `TZ` first there, which Go does not; the parity run leaves `TZ`
//! unset on Windows).

use std::sync::OnceLock;
use std::time::{SystemTime, UNIX_EPOCH};

use jiff::tz::TimeZone;
use jiff::{Timestamp, Zoned};

/// An instant parsed from RFC 3339 text.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Instant {
    pub unix: i64,
    pub nanos: u32,
}

/// Go's zero `time.Time` (0001-01-01T00:00:00Z) in Unix seconds.
const GO_ZERO_UNIX: i64 = -62_135_596_800;

impl Instant {
    /// Go's `Time.IsZero`.
    pub fn is_zero(&self) -> bool {
        self.unix == GO_ZERO_UNIX && self.nanos == 0
    }
}

fn days_in(month: i64, year: i64) -> i64 {
    match month {
        2 if (year % 4 == 0 && year % 100 != 0) || year % 400 == 0 => 29,
        2 => 28,
        4 | 6 | 9 | 11 => 30,
        _ => 31,
    }
}

/// Days from 1970-01-01 to a civil date (proleptic Gregorian).
fn days_from_civil(year: i64, month: i64, day: i64) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y - era * 400;
    let mp = (month + 9) % 12;
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146_097 + doe - 719_468
}

fn digits(text: &[u8], min: i64, max: i64) -> Option<i64> {
    if text.is_empty() || !text.iter().all(u8::is_ascii_digit) {
        return None;
    }
    let value: i64 = std::str::from_utf8(text).ok()?.parse().ok()?;
    (min..=max).contains(&value).then_some(value)
}

/// Go's `time.Parse(time.RFC3339Nano, text)`, including the general
/// parser's comma before fractional seconds.
pub fn parse_rfc3339(text: &str) -> Option<Instant> {
    let s = text.as_bytes();
    if s.len() < 19 {
        return None;
    }
    let year = digits(&s[0..4], 0, 9999)?;
    let month = digits(&s[5..7], 1, 12)?;
    let day = digits(&s[8..10], 1, days_in(month, year))?;
    let hour = digits(&s[11..13], 0, 23)?;
    let minute = digits(&s[14..16], 0, 59)?;
    let second = digits(&s[17..19], 0, 59)?;
    if s[4] != b'-' || s[7] != b'-' || s[10] != b'T' || s[13] != b':' || s[16] != b':' {
        return None;
    }
    let mut rest = &s[19..];
    let mut nanos = 0u32;
    if rest.len() >= 2 && (rest[0] == b'.' || rest[0] == b',') && rest[1].is_ascii_digit() {
        let n = 1 + rest[1..].iter().take_while(|b| b.is_ascii_digit()).count();
        let fraction: String = rest[1..n].iter().take(9).map(|&b| char::from(b)).collect();
        nanos = format!("{fraction:0<9}").parse().ok()?;
        rest = &rest[n..];
    }
    let offset = if rest == b"Z" {
        0
    } else {
        if rest.len() != 6 || !(rest[0] == b'+' || rest[0] == b'-') || rest[3] != b':' {
            return None;
        }
        let hours = digits(&rest[1..3], 0, 23)?;
        let minutes = digits(&rest[4..6], 0, 59)?;
        let offset = (hours * 60 + minutes) * 60;
        if rest[0] == b'-' { -offset } else { offset }
    };
    let days = days_from_civil(year, month, day);
    let unix = days * 86_400 + hour * 3600 + minute * 60 + second - offset;
    Some(Instant { unix, nanos })
}

fn local_zone() -> &'static TimeZone {
    static ZONE: OnceLock<TimeZone> = OnceLock::new();
    ZONE.get_or_init(|| {
        let from_file = |name: &str, path: &str| {
            std::fs::read(path)
                .ok()
                .and_then(|data| TimeZone::tzif(name, &data).ok())
        };
        // Go on Windows reads the system time zone and ignores TZ.
        if cfg!(windows) {
            return TimeZone::try_system().unwrap_or(TimeZone::UTC);
        }
        let zone = match std::env::var_os("TZ") {
            None => from_file("Local", "/etc/localtime"),
            Some(value) => {
                let value = value.to_string_lossy();
                let name = value.strip_prefix(':').unwrap_or(&value);
                if name.starts_with('/') {
                    from_file(name, name)
                } else if name.is_empty() || name == "UTC" {
                    None
                } else {
                    jiff::tz::db().get(name).ok()
                }
            }
        };
        zone.unwrap_or(TimeZone::UTC)
    })
}

fn zoned(unix: i64, nanos: u32) -> Option<Zoned> {
    Timestamp::new(unix, nanos as i32)
        .ok()
        .map(|ts| ts.to_zoned(local_zone().clone()))
}

/// Formats an instant in local time as Go's "2006-01-02 15:04".
pub fn format_local(at: &Instant) -> String {
    match zoned(at.unix, at.nanos) {
        Some(z) => format!(
            "{:04}-{:02}-{:02} {:02}:{:02}",
            z.year(),
            z.month(),
            z.day(),
            z.hour(),
            z.minute()
        ),
        None => String::new(),
    }
}

/// Unix seconds as local "2006-01-02 15:04".
pub fn format_unix_minutes(seconds: i64) -> String {
    format_local(&Instant {
        unix: seconds,
        nanos: 0,
    })
}

const MONTHS: [&str; 12] = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
];

/// The Go CLI's `sinceText`: the time of day within the last 24 hours,
/// "2 Jan" in this year, "2 Jan 2006" before. Missing or zero: empty.
pub fn since_text(text: &str) -> String {
    let Some(at) = parse_rfc3339(text) else {
        return String::new();
    };
    if at.unix <= 0 {
        return String::new();
    }
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    let now_nanos = now.as_nanos() as i128;
    let at_nanos = at.unix as i128 * 1_000_000_000 + at.nanos as i128;
    let (Some(local), Some(local_now)) = (
        zoned(at.unix, at.nanos),
        zoned(now.as_secs() as i64, now.subsec_nanos()),
    ) else {
        return String::new();
    };
    if now_nanos - at_nanos < 24 * 3600 * 1_000_000_000 {
        return format!("{:02}:{:02}", local.hour(), local.minute());
    }
    let month = MONTHS[(local.month() - 1) as usize];
    if local.year() == local_now.year() {
        format!("{} {month}", local.day())
    } else {
        format!("{} {month} {}", local.day(), local.year())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: a time the Go CLI parses printed raw (or the other way round),
    // or the zero time printed as year 1 instead of "-".
    #[test]
    fn rfc3339_rules_match_go() {
        assert_eq!(
            parse_rfc3339("1970-01-01T00:00:00Z"),
            Some(Instant { unix: 0, nanos: 0 })
        );
        assert_eq!(
            parse_rfc3339("2026-11-16T10:00:00.5+10:00").map(|t| t.unix),
            Some(1_794_787_200)
        );
        assert_eq!(
            parse_rfc3339("2026-11-16T10:00:00,25Z").map(|t| t.nanos),
            Some(250_000_000)
        );
        assert!(parse_rfc3339("0001-01-01T00:00:00Z").unwrap().is_zero());
        assert!(parse_rfc3339("2026-02-30T00:00:00Z").is_none());
        assert!(parse_rfc3339("2026-11-16 10:00:00Z").is_none());
        assert!(parse_rfc3339("2026-11-16T10:00:60Z").is_none());
        assert!(parse_rfc3339("2026-11-16T10:00:00").is_none());
        assert!(parse_rfc3339("2026-11-16T10:00:00.Z").is_none());
    }
}
