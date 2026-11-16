//! One CLI run, as the Go CLI's `executeArgs` with Cobra's `ExecuteC`.

use std::io::{self, IsTerminal, Read};
use std::time::Duration;

use crate::cobra::flags::{flag_set, parse, parse_bool};
use crate::cobra::{Args, CmdPath, Command, find, help};
use crate::commands::{self, Ctx, Run};
use crate::errors::CliError;
use crate::output::{Out, object, string};
use crate::settings::Env;

/// Standard input, abstracted so tests can feed `config set-key`.
pub trait Input {
    fn is_terminal(&self) -> bool;
    fn read_limited(&mut self, limit: u64) -> io::Result<Vec<u8>>;
    fn read_password(&mut self) -> io::Result<String>;
}

pub struct Stdin;

impl Input for Stdin {
    fn is_terminal(&self) -> bool {
        io::stdin().is_terminal()
    }
    fn read_limited(&mut self, limit: u64) -> io::Result<Vec<u8>> {
        let mut data = Vec::new();
        io::stdin().take(limit).read_to_end(&mut data)?;
        Ok(data)
    }
    fn read_password(&mut self) -> io::Result<String> {
        rpassword::read_password()
    }
}

const EXEC_HINT: &str = "To pass flags to the remote command, quote it or put it after --, for example: labtether-cli exec nas01 -- df -h";

/// The Go CLI's `requestsJSONOutput`: whether `--json` was asked for, read
/// before Cobra parses anything so even parse errors come out as JSON.
pub fn requests_json(root: &Command, args: &[String]) -> bool {
    let (path, _, _) = find::find(root, args);
    let flags = flag_set(root, &path, false);
    let mut requested = false;
    let mut i = 0;
    while i < args.len() {
        let arg = &args[i];
        if arg == "--" {
            break;
        }
        if let Some(rest) = arg.strip_prefix("--") {
            let (name, value) = match rest.split_once('=') {
                Some((n, v)) => (n, Some(v)),
                None => (rest, None),
            };
            if name == "json" {
                if let Some(parsed) = parse_bool(value.unwrap_or("true")) {
                    requested = parsed;
                }
            } else if value.is_none() && flags.lookup(name).is_some_and(|f| !f.takes_no_value()) {
                i += 1;
            }
        }
        i += 1;
    }
    requested
}

fn is_raw_output(root: &Command, path: &CmdPath) -> bool {
    matches!(
        path.command_path(root).as_str(),
        "labtether-cli files cat" | "labtether-cli docker logs"
    )
}

/// Runs the command named by `args` and returns the exit code. Output goes
/// to `out`; the caller writes it.
pub fn execute(
    args: &[String],
    env: &Env,
    stdin: &mut dyn Input,
    out: &mut Out,
    timeout: Option<Duration>,
) -> i32 {
    let root = commands::root();
    let json = requests_json(&root, args);
    let (path, rest, find_error) = find::find(&root, args);
    let first_word = find::strip_flags(args, &flag_set(&root, &CmdPath::default(), false))
        .into_iter()
        .next();
    let (result, help_ready) = if matches!(
        first_word.as_deref(),
        Some("__complete" | "__completeNoDesc")
    ) {
        (
            Err(CliError::NotInRustBuild(
                "labtether-cli __complete (shell completion answers)".into(),
            )),
            false,
        )
    } else if let Some(err) = find_error {
        (Err(CliError::Usage(err)), false)
    } else {
        (run(&root, &path, &rest, env, stdin, out, timeout), true)
    };
    let Err(err) = result else { return 0 };
    if let CliError::NotInRustBuild(_) = err {
        out.eprint(&format!("Error: {err}\n"));
    } else if json && !is_raw_output(&root, &path) {
        if !out.json_written {
            out.print_json(&object(&[("error", string(&err.to_string()))]));
        }
    } else {
        out.eprint(&format!("Error: {err}\n"));
        if err.shows_usage() {
            out.eprint(&help::usage(&root, &path, help_ready));
            out.eprint("\n");
        }
    }
    err.exit_code()
}

#[allow(clippy::too_many_arguments)]
fn run(
    root: &Command,
    path: &CmdPath,
    rest: &[String],
    env: &Env,
    stdin: &mut dyn Input,
    out: &mut Out,
    timeout: Option<Duration>,
) -> Result<(), CliError> {
    let cmd = path.resolve(root);
    let is_exec = path.command_path(root) == "labtether-cli exec";
    let flags = flag_set(root, path, true);
    let parsed = parse(&flags, rest).map_err(|err| {
        if is_exec && err.starts_with("unknown ") {
            CliError::Usage(format!("{err}. {EXEC_HINT}"))
        } else {
            CliError::Usage(err)
        }
    })?;
    out.eprint(&parsed.notices);
    if parsed.bool("help") {
        if is_exec && !parsed.args.is_empty() {
            return Err(CliError::UsageInRun(format!(
                "-h after the device asks for help and runs nothing. {EXEC_HINT}"
            )));
        }
        out.print(&help::help(root, path));
        return Ok(());
    }
    if let Some(version) = &cmd.version
        && parsed.bool("version")
    {
        out.print(&format!("{} version {version}\n", cmd.name()));
        return Ok(());
    }
    let Some(run) = cmd.run else {
        out.print(&help::help(root, path));
        return Ok(());
    };
    match cmd.args {
        Args::Exact(n) if parsed.args.len() != n => {
            return Err(CliError::Usage(format!(
                "accepts {n} arg(s), received {}",
                parsed.args.len()
            )));
        }
        Args::None if !parsed.args.is_empty() => {
            return Err(CliError::Usage(format!(
                "unknown command {} for {}",
                crate::cobra::quote::quote(&parsed.args[0]),
                crate::cobra::quote::quote(&path.command_path(root))
            )));
        }
        _ => {}
    }
    match run {
        Run::NotYet => Err(CliError::NotInRustBuild(path.command_path(root))),
        Run::Rust(handler) => {
            let json = parsed.bool("json");
            let mut ctx = Ctx {
                root,
                path: path.clone(),
                args: parsed.args.clone(),
                flags: parsed,
                json,
                env,
                out,
                stdin,
                timeout,
            };
            handler(&mut ctx)
        }
    }
}
