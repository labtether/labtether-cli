//! Shell completion: the `completion <shell>` scripts and the hidden
//! `__complete` command they call, as Cobra 1.10.2 has them.
//!
//! The scripts are Cobra's output for this CLI, kept as files beside the
//! code (`../completion/`); `--no-descriptions` changes the one word that
//! names the request command. `__complete` copies Cobra's
//! `getCompletions` for the parts this CLI uses: command names, flag names
//! and the help and completion commands' own completions. No command here
//! completes its arguments or flag values.

use std::io::Write;

use super::Ctx;
use crate::cobra::flags::{FlagSet, Parsed, flag_set, parse};
use crate::cobra::{Args, CmdPath, Command, Flag, find};
use crate::errors::CliError;

const REQUEST: &str = "__complete";
const REQUEST_NO_DESC: &str = "__completeNoDesc";

const DIRECTIVE_NO_SPACE: u32 = 2;
const DIRECTIVE_NO_FILE: u32 = 4;

fn script(ctx: &mut Ctx, text: &str) -> Result<(), CliError> {
    if ctx.flags.bool("no-descriptions") {
        // Each script names the request command once, followed by a space.
        let request = format!(" {REQUEST} ");
        let no_desc = format!(" {REQUEST_NO_DESC} ");
        ctx.out.print(&text.replacen(&request, &no_desc, 1));
    } else {
        ctx.out.print(text);
    }
    Ok(())
}

pub fn bash(ctx: &mut Ctx) -> Result<(), CliError> {
    script(ctx, include_str!("../completion/labtether-cli.bash"))
}

pub fn zsh(ctx: &mut Ctx) -> Result<(), CliError> {
    script(ctx, include_str!("../completion/labtether-cli.zsh"))
}

pub fn fish(ctx: &mut Ctx) -> Result<(), CliError> {
    script(ctx, include_str!("../completion/labtether-cli.fish"))
}

pub fn powershell(ctx: &mut Ctx) -> Result<(), CliError> {
    script(ctx, include_str!("../completion/labtether-cli.ps1"))
}

/// Cobra's hidden `__complete` command. Cobra adds it to the tree only when
/// the arguments call it.
pub fn request_command() -> Command {
    let mut cmd = Command::new(
        "__complete [command-line]",
        "Request shell completion choices for the specified command-line",
    )
    .long(
        "__complete is a special command that is used by the shell completion logic\n\
         to request completion choices for the specified command-line.",
    )
    .aliases(&[REQUEST_NO_DESC])
    .args(Args::Minimum(1))
    .run(super::Run::Rust(request));
    cmd.hidden = true;
    cmd.usage_without_flags = true;
    cmd.disable_flag_parsing = true;
    cmd
}

/// Cobra's `ShellCompDirective.string()`.
fn directive_text(directive: u32) -> &'static str {
    match directive {
        0 => "ShellCompDirectiveDefault",
        DIRECTIVE_NO_SPACE => "ShellCompDirectiveNoSpace",
        DIRECTIVE_NO_FILE => "ShellCompDirectiveNoFileComp",
        _ => "ShellCompDirectiveDefault",
    }
}

/// Go's `%v` for a list of strings.
fn go_list(items: &[String]) -> String {
    format!("[{}]", items.join(" "))
}

/// Cobra's `CompDebug` with stderr: "[Debug] " before the text, also
/// appended to `BASH_COMP_DEBUG_FILE` when that is set.
fn debug(ctx: &mut Ctx, text: &[u8]) {
    let mut line = b"[Debug] ".to_vec();
    line.extend_from_slice(text);
    if let Some(path) = &ctx.env.comp_debug_file {
        let mut options = std::fs::OpenOptions::new();
        options.append(true).create(true);
        #[cfg(unix)]
        std::os::unix::fs::OpenOptionsExt::mode(&mut options, 0o644);
        if let Ok(mut file) = options.open(path) {
            let _ = file.write_all(&line);
        }
    }
    ctx.out.stderr.extend_from_slice(&line);
}

fn request(ctx: &mut Ctx) -> Result<(), CliError> {
    let called_no_desc = ctx.called_as == REQUEST_NO_DESC;
    let (completions, directive, err) = completions(ctx.root, &ctx.args, &mut ctx.out.stderr);
    if let Some(err) = err {
        let mut text = b"[Error] ".to_vec();
        text.extend_from_slice(&err);
        text.push(b'\n');
        debug(ctx, &text);
    }
    let mut no_desc = called_no_desc;
    if !no_desc
        && let Some(on) = ctx
            .env
            .completion_descriptions
            .as_deref()
            .and_then(crate::cobra::flags::parse_bool)
    {
        no_desc = !on;
    }
    for completion in completions {
        let mut choice = completion.as_str();
        if no_desc {
            choice = choice.split('\t').next().unwrap_or_default();
        }
        choice = choice.split('\n').next().unwrap_or_default().trim();
        ctx.out.print(&format!("{choice}\n"));
    }
    ctx.out.print(&format!(":{directive}\n"));
    ctx.out.eprint(&format!(
        "Completion ended with directive: {}\n",
        directive_text(directive)
    ));
    Ok(())
}

/// Cobra's `isFlagArg`.
fn is_flag_arg(arg: &[u8]) -> bool {
    (arg.len() >= 3 && arg.starts_with(b"--"))
        || (arg.len() >= 2 && arg[0] == b'-' && arg[1] != b'-')
}

/// Cobra's `findFlag`: a one-letter name is a shorthand.
fn find_flag<'a>(flags: &'a FlagSet, name: &[u8]) -> Option<&'a Flag> {
    if let [byte] = name {
        let short = flags.lookup_short(*byte as char)?;
        return flags.lookup(short.name);
    }
    flags.lookup(std::str::from_utf8(name).ok()?)
}

/// Cobra's `checkIfFlagCompletion`: the flag whose value is being typed,
/// the arguments without it, the text to complete and an error for a flag
/// the command does not have.
fn flag_value_completion<'a>(
    cmd: &Command,
    flags: &'a FlagSet,
    args: &[String],
    last: &str,
) -> (Option<&'a Flag>, Vec<String>, String, Option<Vec<u8>>) {
    if cmd.disable_flag_parsing {
        return (None, args.to_vec(), last.to_string(), None);
    }
    let bytes = last.as_bytes();
    let mut name: Vec<u8> = Vec::new();
    let mut trimmed = args.to_vec();
    let mut typed = last.to_string();
    let mut with_equal = false;
    if bytes.first() == Some(&b'-') {
        let Some(index) = last.find('=') else {
            return (None, args.to_vec(), last.to_string(), None);
        };
        name = if bytes[..index].starts_with(b"--") {
            bytes[2..index].to_vec()
        } else {
            bytes[index - 1..index].to_vec()
        };
        typed = last[index + 1..].to_string();
        with_equal = true;
    }
    if name.is_empty()
        && let Some(previous) = args.last()
    {
        let prev = previous.as_bytes();
        if is_flag_arg(prev) && !previous.contains('=') {
            name = if prev.starts_with(b"--") {
                prev[2..].to_vec()
            } else {
                prev[prev.len() - 1..].to_vec()
            };
            trimmed = args[..args.len() - 1].to_vec();
        }
    }
    if name.is_empty() {
        return (None, trimmed, typed, None);
    }
    let Some(flag) = find_flag(flags, &name) else {
        let mut err = format!("Subcommand '{}' does not support flag '", cmd.name()).into_bytes();
        err.extend_from_slice(&name);
        err.push(b'\'');
        return (None, args.to_vec(), last.to_string(), Some(err));
    };
    if !with_equal && flag.takes_no_value() {
        return (None, args.to_vec(), typed, None);
    }
    (Some(flag), trimmed, typed, None)
}

/// Cobra's `ParseFlags` during completion: pflag's notices (a deprecated
/// flag) go into the command's buffer, and the whole buffer is printed
/// after each parse that succeeds.
fn parse_flags(
    flags: &FlagSet,
    args: &[String],
    buffer: &mut String,
    stderr: &mut Vec<u8>,
) -> Result<Parsed, String> {
    let parsed = parse(flags, args)?;
    buffer.push_str(&parsed.notices);
    stderr.extend_from_slice(buffer.as_bytes());
    Ok(parsed)
}

/// `--name` and `-x` choices for a flag that starts with `typed`.
fn flag_names(flag: &Flag, typed: &str, out: &mut Vec<String>) {
    if flag.hidden() {
        return;
    }
    let long = format!("--{}", flag.name);
    if long.starts_with(typed) {
        out.push(format!("{long}\t{}", flag.usage));
    }
    if let Some(short) = flag.shorthand {
        let short = format!("-{short}");
        if short.starts_with(typed) {
            out.push(format!("{short}\t{}", flag.usage));
        }
    }
}

/// The commands under `cmd` a user can pick, with the root's help.
fn command_names(cmd: &Command, typed: &str, out: &mut Vec<String>) -> bool {
    let mut any = false;
    for child in &cmd.children {
        if child.is_available() || child.is_help {
            if child.name().starts_with(typed) {
                out.push(format!("{}\t{}", child.name(), child.short));
            }
            any = true;
        }
    }
    any
}

/// Cobra's `getCompletions`: the choices, the directive and an error.
fn completions(
    root: &Command,
    args: &[String],
    stderr: &mut Vec<u8>,
) -> (Vec<String>, u32, Option<Vec<u8>>) {
    let (to_complete, trimmed) = args.split_last().expect("at least one argument");
    let (path, found_args, err) = find::find(root, trimmed);
    if err.is_some() {
        let text = format!(
            "unable to find a command for arguments: {}",
            go_list(trimmed)
        );
        return (Vec::new(), 0, Some(text.into_bytes()));
    }
    let cmd = path.resolve(root);
    let flags = flag_set(root, &path, true);
    let (flag, mut final_args, typed, flag_err) =
        flag_value_completion(cmd, &flags, &found_args, to_complete);
    let mut buffer = String::new();
    let (parsed, flag_completion) = if cmd.disable_flag_parsing {
        (Parsed::default(), true)
    } else {
        let mut with_end = final_args.clone();
        with_end.push("--".into());
        let with_end = parse_flags(&flags, &with_end, &mut buffer, stderr)
            .map(|p| p.args.len())
            .unwrap_or(0);
        match parse_flags(&flags, &final_args, &mut buffer, stderr) {
            Ok(parsed) => {
                let count = parsed.args.len();
                (parsed, with_end <= count)
            }
            Err(e) => {
                let text = format!(
                    "Error while parsing flags from args {}: {e}",
                    go_list(&final_args)
                );
                return (Vec::new(), 0, Some(text.into_bytes()));
            }
        }
    };
    if let Some(err) = flag_err
        && flag_completion
    {
        return (Vec::new(), 0, Some(err));
    }
    if parsed.changed.contains("help") || parsed.changed.contains("version") {
        return (Vec::new(), DIRECTIVE_NO_FILE, None);
    }
    if !cmd.disable_flag_parsing {
        final_args = parsed.args.clone();
    }
    let mut out = Vec::new();
    let mut directive = 0;
    if flag.is_none() && typed.starts_with('-') && !typed.contains('=') && flag_completion {
        for f in flags.inherited.iter().chain(flags.local.iter()) {
            if !parsed.changed.contains(f.name) {
                flag_names(f, &typed, &mut out);
            }
        }
        directive = if out.len() == 1 && out[0].ends_with('=') {
            DIRECTIVE_NO_SPACE
        } else {
            DIRECTIVE_NO_FILE
        };
        if !cmd.disable_flag_parsing {
            return (out, directive, None);
        }
    } else if flag.is_none() {
        let own_flag_set = cmd
            .flags
            .iter()
            .map(|f| f.name)
            .chain(["help", "version"])
            .any(|name| parsed.changed.contains(name));
        if final_args.is_empty() && !own_flag_set && command_names(cmd, &typed, &mut out) {
            directive = DIRECTIVE_NO_FILE;
        }
    }
    if flag.is_some() && flag_completion {
        // No flag in this CLI completes its value.
        return (out, directive, None);
    }
    // The commands' own argument completion (Cobra's ValidArgsFunction).
    match path.command_path(root).as_str() {
        "labtether-cli help" => {
            let (target, _, err) = find::find(root, &final_args);
            if err.is_none() {
                command_names(target.resolve(root), &typed, &mut out);
            }
            directive = DIRECTIVE_NO_FILE;
        }
        name if name == "labtether-cli completion"
            || name.starts_with("labtether-cli completion ") =>
        {
            directive = DIRECTIVE_NO_FILE;
        }
        _ => {}
    }
    (out, directive, None)
}

/// The command path Cobra adds `__complete` to: the root, when the
/// arguments call it. `called_as` is the name used.
pub fn requested(root: &Command, args: &[String]) -> Option<(Command, String)> {
    let mut with = root.clone();
    let mut children = std::mem::take(&mut with.children);
    children.push(request_command());
    with = with.children(children);
    let (path, _, err) = find::find(&with, args);
    if err.is_some() || path.0.len() != 1 || path.resolve(&with).name() != REQUEST {
        return None;
    }
    let first = find::strip_flags(args, &flag_set(&with, &CmdPath::default(), false))
        .into_iter()
        .next()
        .unwrap_or_default();
    Some((with, first))
}
