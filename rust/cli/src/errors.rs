//! CLI errors. The exit code comes from the variant; the Go CLI reaches the
//! same codes by matching text in `cmd/exit_codes.go`.

use std::fmt;

use labtether_cli_client::ClientError;

/// Exit codes, as listed in the CLI guide.
pub const EXIT_FAILED: i32 = 1;
pub const EXIT_AUTH: i32 = 2;
pub const EXIT_NOT_FOUND: i32 = 3;
pub const EXIT_CONFIG: i32 = 4;

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum CliError {
    /// A bad command, flag or argument found while parsing. Shows usage.
    Usage(String),
    /// A bad argument a command finds itself. Shows usage too.
    UsageInRun(String),
    /// The CLI's own settings: Hub URL, key, key file, CA file, config file.
    Config(String),
    /// The Hub answered with an error status.
    Hub { status: u16, message: String },
    /// An object the CLI looked up itself and did not find.
    NotFound(String),
    /// A command this Rust build does not have yet.
    NotInRustBuild(String),
    /// Anything else: transport, decode, size limit.
    Failed(String),
}

impl fmt::Display for CliError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CliError::Hub { status, message } => write!(f, "{message} (status {status})"),
            CliError::NotInRustBuild(path) => write!(
                f,
                "{path} is not in the Rust build yet; use the Go labtether-cli"
            ),
            CliError::Usage(m)
            | CliError::UsageInRun(m)
            | CliError::Config(m)
            | CliError::NotFound(m)
            | CliError::Failed(m) => f.write_str(m),
        }
    }
}

impl CliError {
    pub fn exit_code(&self) -> i32 {
        match self {
            CliError::Config(_) => EXIT_CONFIG,
            CliError::Hub {
                status: 401 | 403, ..
            } => EXIT_AUTH,
            CliError::Hub {
                status: 404 | 409, ..
            }
            | CliError::NotFound(_) => EXIT_NOT_FOUND,
            _ => EXIT_FAILED,
        }
    }

    /// Whether usage text follows the error, as the Go CLI's `showsUsage`.
    pub fn shows_usage(&self) -> bool {
        matches!(self, CliError::Usage(_) | CliError::UsageInRun(_))
    }
}

impl From<ClientError> for CliError {
    fn from(err: ClientError) -> Self {
        match err {
            ClientError::Config(m) => CliError::Config(m),
            ClientError::Status { status, message } => CliError::Hub { status, message },
            ClientError::Failed(m) => CliError::Failed(m),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Fault: a refused key, a missing object or a settings problem exited 1,
    // so scripts could not tell them apart (the CLI guide promises 2, 3, 4).
    #[test]
    fn each_situation_has_its_exit_code() {
        let hub = |status| CliError::Hub {
            status,
            message: "x".into(),
        };
        assert_eq!(hub(401).exit_code(), 2);
        assert_eq!(hub(403).exit_code(), 2);
        assert_eq!(hub(404).exit_code(), 3);
        assert_eq!(hub(409).exit_code(), 3);
        assert_eq!(hub(500).exit_code(), 1);
        assert_eq!(hub(400).exit_code(), 1);
        assert_eq!(
            CliError::NotFound("connector \"x\" not found".into()).exit_code(),
            3
        );
        assert_eq!(
            CliError::Config("hub host not configured".into()).exit_code(),
            4
        );
        assert_eq!(
            CliError::Usage("accepts 1 arg(s), received 0".into()).exit_code(),
            1
        );
        assert_eq!(
            CliError::Failed("status 404: not json".into()).exit_code(),
            1
        );
        assert_ne!(CliError::NotInRustBuild("exec".into()).exit_code(), 0);
    }

    // Fault: the code came from message text, so a Hub message that happens
    // to contain "(status 404)" turned a 500 into exit 3.
    #[test]
    fn exit_code_ignores_message_text() {
        let err = CliError::Hub {
            status: 500,
            message: "upstream said (status 404)".into(),
        };
        assert_eq!(err.exit_code(), 1);
        assert_eq!(CliError::Failed("x (status 401)".into()).exit_code(), 1);
    }
}
