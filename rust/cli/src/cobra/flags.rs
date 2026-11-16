//! pflag's flag set and parser, with its error messages.

use std::collections::{HashMap, HashSet};

use crate::cobra::quote::{quote, quote_rune};

use super::{CmdPath, Command, Flag, FlagKind};

/// The flags one command sees: its own, its persistent ones, Cobra's help
/// and version flags once they are added, and its parents' persistent ones.
#[derive(Clone, Debug)]
pub struct FlagSet {
    pub local: Vec<Flag>,
    pub inherited: Vec<Flag>,
}

/// Builds the flag set for `path`. `with_help` adds Cobra's help flag (and
/// the root's version flag), which Cobra adds only when it runs or shows
/// help for a command.
pub fn flag_set(root: &Command, path: &CmdPath, with_help: bool) -> FlagSet {
    let chain = path.chain(root);
    let cmd = *chain.last().expect("root");
    let mut local: Vec<Flag> = cmd
        .flags
        .iter()
        .chain(cmd.persistent.iter())
        .cloned()
        .collect();
    if with_help {
        let mut help = Flag::bool("help", &format!("help for {}", cmd.name()));
        help.shorthand = Some('h');
        local.push(help);
        if let Some(_version) = &cmd.version {
            let mut version = Flag::bool("version", &format!("version for {}", cmd.name()));
            version.shorthand = Some('v');
            local.push(version);
        }
    }
    let inherited = chain[..chain.len() - 1]
        .iter()
        .flat_map(|c| c.persistent.iter().cloned())
        .collect();
    local.sort_by(|a, b| a.name.cmp(b.name));
    let mut set = FlagSet { local, inherited };
    set.inherited.sort_by(|a, b| a.name.cmp(b.name));
    set
}

impl FlagSet {
    pub fn all(&self) -> impl Iterator<Item = &Flag> {
        self.local.iter().chain(self.inherited.iter())
    }

    pub fn lookup(&self, name: &str) -> Option<&Flag> {
        self.all().find(|f| f.name == name)
    }

    pub fn lookup_short(&self, c: char) -> Option<&Flag> {
        self.all().find(|f| f.shorthand == Some(c))
    }
}

/// Parsed flag values and the positional arguments left over.
#[derive(Clone, Debug, Default)]
pub struct Parsed {
    values: HashMap<&'static str, String>,
    pub changed: HashSet<&'static str>,
    pub args: Vec<String>,
    /// pflag's deprecation notices, printed to stderr when parsing succeeds.
    pub notices: String,
}

impl Parsed {
    pub fn string(&self, name: &str) -> String {
        self.values.get(name).cloned().unwrap_or_default()
    }

    pub fn bool(&self, name: &str) -> bool {
        self.values.get(name).is_some_and(|v| v == "true")
    }

    pub fn int(&self, name: &str) -> i64 {
        self.values
            .get(name)
            .and_then(|v| v.parse().ok())
            .unwrap_or(0)
    }
}

/// Go's `strconv.ParseBool`.
pub fn parse_bool(text: &str) -> Option<bool> {
    match text {
        "1" | "t" | "T" | "TRUE" | "true" | "True" => Some(true),
        "0" | "f" | "F" | "FALSE" | "false" | "False" => Some(false),
        _ => None,
    }
}

/// The Go CLI's `strictDecimalInt`: digits only, no sign or spaces.
fn strict_decimal(text: &str) -> Option<i64> {
    if text.is_empty() || !text.bytes().all(|b| b.is_ascii_digit()) {
        return None;
    }
    text.parse().ok()
}

fn set_value(flag: &Flag, value: &str, parsed: &mut Parsed) -> Result<(), String> {
    let stored = match &flag.kind {
        FlagKind::String => Ok(value.to_string()),
        FlagKind::Bool => parse_bool(value).map(|b| b.to_string()).ok_or_else(|| {
            format!(
                "strconv.ParseBool: parsing {}: invalid syntax",
                quote(value)
            )
        }),
        FlagKind::BoundedInt {
            name,
            unit,
            min,
            max,
        } => match strict_decimal(value) {
            Some(n) if (*min..=*max).contains(&n) => Ok(n.to_string()),
            _ if unit.is_empty() => Err(format!("{name} must be between {min} and {max}")),
            _ => Err(format!("{name} must be between {min} and {max} {unit}")),
        },
    };
    match stored {
        Ok(stored) => {
            parsed.values.insert(flag.name, stored);
            parsed.changed.insert(flag.name);
            if let Some(notice) = flag.deprecated {
                parsed.notices.push_str(&format!(
                    "Flag --{} has been deprecated, {notice}\n",
                    flag.name
                ));
            }
            Ok(())
        }
        Err(cause) => {
            let shown = match flag.shorthand {
                Some(c) => format!("-{c}, --{}", flag.name),
                None => format!("--{}", flag.name),
            };
            Err(format!(
                "invalid argument {} for {} flag: {cause}",
                quote(value),
                quote(&shown)
            ))
        }
    }
}

/// pflag's `Parse` with interspersed flags. Errors are pflag's messages.
pub fn parse(set: &FlagSet, input: &[String]) -> Result<Parsed, String> {
    let mut parsed = Parsed::default();
    for flag in set.all() {
        parsed.values.insert(flag.name, flag.default.clone());
    }
    let mut args = input.iter();
    while let Some(arg) = args.next() {
        if !arg.starts_with('-') || arg.len() == 1 {
            parsed.args.push(arg.clone());
            continue;
        }
        if arg == "--" {
            parsed.args.extend(args.by_ref().cloned());
            break;
        }
        if let Some(name) = arg.strip_prefix("--") {
            if name.starts_with('-') || name.starts_with('=') {
                return Err(format!("bad flag syntax: {arg}"));
            }
            let (name, inline) = match name.split_once('=') {
                Some((n, v)) => (n, Some(v.to_string())),
                None => (name, None),
            };
            let Some(flag) = set.lookup(name) else {
                return Err(format!("unknown flag: --{name}"));
            };
            let value = match inline {
                Some(v) => v,
                None if flag.takes_no_value() => "true".to_string(),
                None => match args.next() {
                    Some(v) => v.clone(),
                    None => return Err(format!("flag needs an argument: --{name}")),
                },
            };
            set_value(flag, &value, &mut parsed)?;
            continue;
        }
        let mut rest = &arg[1..];
        while let Some(c) = rest.chars().next() {
            let current = rest;
            let after = &rest[c.len_utf8()..];
            let Some(flag) = set.lookup_short(c) else {
                return Err(format!(
                    "unknown shorthand flag: {} in -{rest}",
                    quote_rune(c)
                ));
            };
            let value = if let Some(v) = after.strip_prefix('=').filter(|_| after.len() > 1) {
                rest = "";
                v.to_string()
            } else if flag.takes_no_value() {
                rest = after;
                "true".to_string()
            } else if !after.is_empty() {
                rest = "";
                after.to_string()
            } else {
                rest = "";
                match args.next() {
                    Some(v) => v.clone(),
                    None => {
                        return Err(format!(
                            "flag needs an argument: {} in -{current}",
                            quote_rune(c)
                        ));
                    }
                }
            };
            set_value(flag, &value, &mut parsed)?;
        }
    }
    Ok(parsed)
}
