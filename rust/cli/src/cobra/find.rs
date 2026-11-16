//! Cobra's `Find`: which command the arguments name, and the "unknown
//! command" error with its suggestions.

use crate::cobra::quote::quote;

use super::flags::{FlagSet, flag_set};
use super::{Args, CmdPath, Command};

fn has_no_opt_default(name: &str, flags: &FlagSet) -> bool {
    flags.lookup(name).is_some_and(|f| f.takes_no_value())
}

fn short_has_no_opt_default(name: &str, flags: &FlagSet) -> bool {
    name.chars()
        .next()
        .and_then(|c| flags.lookup_short(c))
        .is_some_and(|f| f.takes_no_value())
}

/// Whether `arg` is a flag that takes the next argument as its value, by
/// Cobra's rule (an unknown flag counts as one).
fn takes_next(arg: &str, flags: &FlagSet) -> bool {
    if arg.contains('=') {
        return false;
    }
    if let Some(name) = arg.strip_prefix("--") {
        return !has_no_opt_default(name, flags);
    }
    arg.starts_with('-') && arg.len() == 2 && !short_has_no_opt_default(&arg[1..], flags)
}

/// Cobra's `stripFlags`: the words that are not flags or flag values.
pub fn strip_flags(args: &[String], flags: &FlagSet) -> Vec<String> {
    let mut commands = Vec::new();
    let mut rest = args;
    while let Some((first, tail)) = rest.split_first() {
        rest = tail;
        if first == "--" {
            break;
        }
        if takes_next(first, flags) {
            if rest.len() <= 1 {
                break;
            }
            rest = &rest[1..];
            continue;
        }
        if !first.is_empty() && !first.starts_with('-') {
            commands.push(first.clone());
        }
    }
    commands
}

/// Cobra's `argsMinusFirstX`: removes the first `x` that is not a flag value.
fn args_minus_first(args: &[String], x: &str, flags: &FlagSet) -> Vec<String> {
    let mut pos = 0;
    while pos < args.len() {
        let s = &args[pos];
        if s == "--" {
            break;
        }
        if takes_next(s, flags) {
            pos += 2;
            continue;
        }
        if !s.starts_with('-') && s == x {
            let mut out = args.to_vec();
            out.remove(pos);
            return out;
        }
        pos += 1;
    }
    args.to_vec()
}

/// Cobra's `ld`: Levenshtein distance over bytes, ignoring ASCII case.
fn distance(a: &str, b: &str) -> usize {
    let (a, b) = (a.to_lowercase().into_bytes(), b.to_lowercase().into_bytes());
    let mut row: Vec<usize> = (0..=b.len()).collect();
    for i in 1..=a.len() {
        let mut previous = row[0];
        row[0] = i;
        for j in 1..=b.len() {
            let current = row[j];
            row[j] = if a[i - 1] == b[j - 1] {
                previous
            } else {
                previous.min(row[j]).min(row[j - 1]) + 1
            };
            previous = current;
        }
    }
    row[b.len()]
}

/// Cobra's `findSuggestions` for a mistyped command.
pub fn suggestions(cmd: &Command, typed: &str) -> String {
    let names: Vec<&str> = cmd
        .children
        .iter()
        .filter(|c| c.is_available())
        .filter(|c| {
            distance(typed, c.name()) <= 2
                || c.name().to_lowercase().starts_with(&typed.to_lowercase())
        })
        .map(Command::name)
        .collect();
    if names.is_empty() {
        return String::new();
    }
    let mut out = String::from("\n\nDid you mean this?\n");
    for name in names {
        out.push_str(&format!("\t{name}\n"));
    }
    out
}

/// Cobra's `Find`: the command, the arguments left for it, and the
/// "unknown command" error for the root.
pub fn find(root: &Command, args: &[String]) -> (CmdPath, Vec<String>, Option<String>) {
    let mut path = CmdPath::default();
    let mut rest = args.to_vec();
    loop {
        let cmd = path.resolve(root);
        let flags = flag_set(root, &path, false);
        let words = strip_flags(&rest, &flags);
        let Some(next) = words.first() else { break };
        let Some(index) = cmd.child(next) else { break };
        rest = args_minus_first(&rest, next, &flags);
        path = path.child(index);
    }
    let cmd = path.resolve(root);
    let mut err = None;
    if cmd.args == Args::Legacy && !cmd.children.is_empty() && path.is_root() {
        let flags = flag_set(root, &path, false);
        if let Some(first) = strip_flags(&rest, &flags).first() {
            err = Some(format!(
                "unknown command {} for {}{}",
                quote(first),
                quote(&path.command_path(root)),
                suggestions(cmd, first)
            ));
        }
    }
    (path, rest, err)
}
