//! Typed client errors. The CLI picks its exit code from the variant, never
//! from the message; the message is the Go CLI's exact text.

use std::fmt;

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ClientError {
    /// A settings problem found before any request: the Hub URL or the CA
    /// file. The CLI exits 4.
    Config(String),
    /// The Hub answered with an error status and a JSON body. Prints as
    /// "<message> (status <code>)", like the Go CLI.
    Status { status: u16, message: String },
    /// Anything else: the connection, TLS, a body that is not JSON, a size
    /// limit. The CLI exits 1.
    Failed(String),
}

impl fmt::Display for ClientError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ClientError::Config(message) | ClientError::Failed(message) => f.write_str(message),
            ClientError::Status { status, message } => write!(f, "{message} (status {status})"),
        }
    }
}

impl std::error::Error for ClientError {}

/// Go's text for an operating system error number, as `syscall.Errno`
/// prints it (lower case, no "(os error N)").
pub fn errno_text(err: &std::io::Error) -> String {
    let known: Option<&str> = match err.raw_os_error() {
        #[cfg(unix)]
        Some(code) => match code {
            1 => Some("operation not permitted"),
            2 => Some("no such file or directory"),
            5 => Some("input/output error"),
            6 => Some("no such device or address"),
            13 => Some("permission denied"),
            17 => Some("file exists"),
            20 => Some("not a directory"),
            21 => Some("is a directory"),
            28 => Some("no space left on device"),
            30 => Some("read-only file system"),
            36 if cfg!(target_os = "linux") => Some("file name too long"),
            40 if cfg!(target_os = "linux") => Some("too many levels of symbolic links"),
            62 if cfg!(target_os = "macos") => Some("too many levels of symbolic links"),
            63 if cfg!(target_os = "macos") => Some("file name too long"),
            101 if cfg!(target_os = "linux") => Some("network is unreachable"),
            111 if cfg!(target_os = "linux") => Some("connection refused"),
            113 if cfg!(target_os = "linux") => Some("no route to host"),
            61 if cfg!(target_os = "macos") => Some("connection refused"),
            51 if cfg!(target_os = "macos") => Some("network is unreachable"),
            65 if cfg!(target_os = "macos") => Some("no route to host"),
            _ => None,
        },
        _ => None,
    };
    match known {
        Some(text) => text.to_string(),
        None => {
            let text = err.to_string();
            let text = text.split(" (os error").next().unwrap_or(&text).to_string();
            // Go prints Windows system messages as Windows writes them.
            if cfg!(windows) {
                return text;
            }
            let mut chars = text.chars();
            match chars.next() {
                Some(first) => first.to_lowercase().chain(chars).collect(),
                None => text,
            }
        }
    }
}

/// Go's `*os.PathError` text: "<op> <path>: <errno text>".
pub fn path_error(op: &str, path: &str, err: &std::io::Error) -> String {
    format!("{op} {path}: {}", errno_text(err))
}
