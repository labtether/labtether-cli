//! Cobra's default help and usage text and pflag's flag table.

use crate::cobra::quote::quote;

use super::flags::{FlagSet, flag_set};
use super::{CmdPath, Command, Flag, FlagKind};

/// pflag's `defaultIsZeroValue` for the flag kinds the CLI uses.
fn default_is_zero(flag: &Flag) -> bool {
    match flag.kind {
        FlagKind::Bool => flag.default == "false" || flag.default.is_empty(),
        FlagKind::String => flag.default.is_empty(),
        FlagKind::BoundedInt { .. } => {
            matches!(flag.default.as_str(), "false" | "<nil>" | "" | "0")
        }
    }
}

/// pflag's `FlagUsagesWrapped(0)`.
pub fn flag_usages(flags: &[Flag]) -> String {
    let mut lines = Vec::new();
    let mut max = 0;
    for flag in flags.iter().filter(|f| !f.hidden()) {
        let mut line = match flag.shorthand {
            Some(c) => format!("  -{c}, --{}", flag.name),
            None => format!("      --{}", flag.name),
        };
        if flag.kind != FlagKind::Bool {
            line.push(' ');
            line.push_str(flag.type_name());
        }
        max = max.max(line.len() + 1);
        let mut usage = flag.usage.clone();
        if !default_is_zero(flag) {
            if flag.kind == FlagKind::String {
                usage.push_str(&format!(" (default {})", quote(&flag.default)));
            } else {
                usage.push_str(&format!(" (default {})", flag.default));
            }
        }
        lines.push((line, usage));
    }
    let mut out = String::new();
    for (prefix, usage) in lines {
        let spacing = " ".repeat(max - prefix.len());
        out.push_str(&format!("{prefix} {spacing} {usage}\n"));
    }
    out
}

fn trim_right(text: &str) -> &str {
    text.trim_end()
}

/// Cobra's `UseLine`.
pub fn use_line(root: &Command, path: &CmdPath, flags: &FlagSet) -> String {
    let cmd = path.resolve(root);
    let mut line = if path.is_root() {
        cmd.usage.to_string()
    } else {
        let parent = CmdPath(path.0[..path.0.len() - 1].to_vec());
        format!("{} {}", parent.command_path(root), cmd.usage)
    };
    if !cmd.usage_without_flags && flags.all().any(|f| !f.hidden()) && !line.contains("[flags]") {
        line.push_str(" [flags]");
    }
    line
}

/// Cobra's default usage text with aliases left out, as the Go CLI's
/// usage function does.
pub fn usage(root: &Command, path: &CmdPath, with_help: bool) -> String {
    let cmd = path.resolve(root);
    let flags = flag_set(root, path, with_help);
    let command_path = path.command_path(root);
    let mut out = String::from("Usage:");
    if cmd.runnable() {
        out.push_str(&format!("\n  {}", use_line(root, path, &flags)));
    }
    let has_children = cmd.has_available_children();
    if has_children {
        out.push_str(&format!("\n  {command_path} [command]"));
    }
    if !cmd.example.is_empty() {
        out.push_str(&format!("\n\nExamples:\n{}", cmd.example));
    }
    if has_children {
        out.push_str("\n\nAvailable Commands:");
        let padding = cmd.name_padding();
        for child in cmd
            .children
            .iter()
            .filter(|c| c.is_available() || c.is_help)
        {
            out.push_str(&format!("\n  {:<padding$} {}", child.name(), child.short));
        }
    }
    if flags.local.iter().any(|f| !f.hidden()) {
        out.push_str(&format!(
            "\n\nFlags:\n{}",
            trim_right(&flag_usages(&flags.local))
        ));
    }
    if flags.inherited.iter().any(|f| !f.hidden()) {
        out.push_str(&format!(
            "\n\nGlobal Flags:\n{}",
            trim_right(&flag_usages(&flags.inherited))
        ));
    }
    if has_children {
        out.push_str(&format!(
            "\n\nUse \"{command_path} [command] --help\" for more information about a command."
        ));
    }
    out.push('\n');
    out
}

/// Cobra's default help text: the long (or short) description, then usage.
pub fn help(root: &Command, path: &CmdPath) -> String {
    let cmd = path.resolve(root);
    let text = if cmd.long.is_empty() {
        &cmd.short
    } else {
        &cmd.long
    };
    let text = trim_right(text);
    let mut out = String::new();
    if !text.is_empty() {
        out.push_str(text);
        out.push_str("\n\n");
    }
    if cmd.runnable() || !cmd.children.is_empty() {
        out.push_str(&usage(root, path, true));
    }
    out
}
