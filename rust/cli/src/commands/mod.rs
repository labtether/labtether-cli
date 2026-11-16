//! The command tree: every command of the Go CLI with its usage, aliases,
//! flags, argument rule and handler.

mod actions;
pub mod common;
pub mod complete;
mod config;
mod exec;
mod groups;
mod inventory;
mod meta;
mod monitoring;
mod platforms;
mod remote;
mod targets;

use labtether_cli_client::Client;

use crate::cobra::flags::Parsed;
use crate::cobra::{CmdPath, Command, Flag};
use crate::errors::CliError;
use crate::output::Out;
use crate::settings::{self, Env, Flags};

pub type Handler = fn(&mut Ctx) -> Result<(), CliError>;

#[derive(Clone, Copy, Debug)]
pub enum Run {
    /// A command the Rust build runs.
    Rust(Handler),
}

impl std::fmt::Debug for Ctx<'_> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Ctx")
            .field("args", &self.args)
            .field("json", &self.json)
            .finish()
    }
}

/// What a handler gets: its arguments and flags, the settings sources and
/// the output buffers.
pub struct Ctx<'a> {
    pub root: &'a Command,
    pub path: CmdPath,
    pub args: Vec<String>,
    pub flags: Parsed,
    pub json: bool,
    /// The command word as typed (an alias or the name), Cobra's
    /// `CalledAs`.
    pub called_as: String,
    pub env: &'a Env,
    pub out: &'a mut Out,
    pub stdin: &'a mut dyn crate::app::Input,
    /// Tests shorten the request deadline.
    pub timeout: Option<std::time::Duration>,
}

impl Ctx<'_> {
    pub fn connection_flags(&self) -> Flags {
        Flags {
            host: self.flags.string("host"),
            api_key: self.flags.string("api-key"),
            api_key_file: self.flags.string("api-key-file"),
            ca_file: self.flags.string("tls-ca-file"),
        }
    }

    /// The Go CLI's `newClient`: settings errors exit 4.
    pub fn client(&self) -> Result<Client, CliError> {
        let mut client = settings::new_client(self.env, &self.connection_flags())?;
        if let Some(timeout) = self.timeout {
            client.timeout = timeout;
        }
        Ok(client)
    }

    /// A client with a longer request deadline, as the Go CLI sets for
    /// package and Proxmox actions. Tests keep their own deadline.
    pub fn client_with(&self, timeout: std::time::Duration) -> Result<Client, CliError> {
        let mut client = self.client()?;
        if self.timeout.is_none() {
            client.timeout = timeout;
        }
        Ok(client)
    }
}

/// A runnable leaf with a Rust handler.
pub fn leaf(usage: &'static str, short: &str, handler: Handler) -> Command {
    Command::new(usage, short).run(Run::Rust(handler))
}

/// The build version, `dev` unless the build sets `LABTETHER_CLI_VERSION`,
/// as the Go build sets `cmd.version` with `-ldflags`.
pub fn version() -> &'static str {
    option_env!("LABTETHER_CLI_VERSION").unwrap_or("dev")
}

/// The whole tree, as Cobra has it once `Execute` adds the help and
/// completion commands.
pub fn root() -> Command {
    let mut api_key = Flag::string("api-key", "", "API key (overrides config)");
    api_key.deprecated = Some(
        "command-line secrets are visible in shell history and process listings; use --api-key-file or LABTETHER_API_KEY",
    );
    let mut root = Command::new("labtether-cli", "Run your LabTether hub from the terminal")
        .long(
            "labtether-cli works with your LabTether hub from a shell or a script. \
             Add --json to a command to print the hub's data as JSON.",
        )
        .example("  labtether-cli devices list\n  labtether-cli exec nas01 -- df -h\n  labtether-cli issues list --json");
    root.version = Some(version().to_string());
    root.persistent = vec![
        Flag::string("host", "", "Hub URL (overrides the config file)"),
        api_key,
        Flag::string("api-key-file", "", "Read API key from a protected file"),
        Flag::string(
            "tls-ca-file",
            "",
            "Trust additional PEM CA certificates for the hub",
        ),
        Flag::bool("json", "Print the result as JSON"),
    ];
    let mut children = Vec::new();
    children.extend(inventory::commands());
    children.extend(targets::commands());
    children.extend(monitoring::commands());
    children.extend(groups::commands());
    children.extend(platforms::commands());
    children.push(config::command());
    children.extend(meta::commands());
    root.children(children)
}
