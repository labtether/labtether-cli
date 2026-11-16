//! A copy of the parts of Cobra 1.10 and pflag 1.0 that the Go CLI uses, so
//! command lookup, flag parsing, help, usage and their error messages come
//! out byte for byte the same.

pub mod find;
pub mod flags;
pub mod help;
pub mod quote;

use crate::commands::Run;

/// How a command checks its positional arguments.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Args {
    /// No rule (Cobra's `Args == nil`): checked only for the root command,
    /// when it is found.
    Legacy,
    Exact(usize),
    None,
    Arbitrary,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FlagKind {
    Bool,
    String,
    /// The Go CLI's `boundedIntValue`: a whole number within a range.
    BoundedInt {
        name: &'static str,
        unit: &'static str,
        min: i64,
        max: i64,
    },
}

#[derive(Clone, Debug)]
pub struct Flag {
    pub name: &'static str,
    pub shorthand: Option<char>,
    pub usage: String,
    pub kind: FlagKind,
    pub default: String,
    /// pflag's `MarkDeprecated`: hidden in help, a notice when used.
    pub deprecated: Option<&'static str>,
}

impl Flag {
    pub fn bool(name: &'static str, usage: &str) -> Flag {
        Flag {
            name,
            shorthand: None,
            usage: usage.to_string(),
            kind: FlagKind::Bool,
            default: "false".into(),
            deprecated: None,
        }
    }

    pub fn string(name: &'static str, default: &str, usage: &str) -> Flag {
        Flag {
            name,
            shorthand: None,
            usage: usage.to_string(),
            kind: FlagKind::String,
            default: default.into(),
            deprecated: None,
        }
    }

    pub fn bounded_int(
        name: &'static str,
        default: i64,
        unit: &'static str,
        min: i64,
        max: i64,
        usage: &str,
    ) -> Flag {
        Flag {
            name,
            shorthand: None,
            usage: usage.to_string(),
            kind: FlagKind::BoundedInt {
                name,
                unit,
                min,
                max,
            },
            default: default.to_string(),
            deprecated: None,
        }
    }

    /// pflag's `NoOptDefVal`: a bool flag needs no value.
    pub fn takes_no_value(&self) -> bool {
        self.kind == FlagKind::Bool
    }

    pub fn hidden(&self) -> bool {
        self.deprecated.is_some()
    }

    /// pflag's `Value.Type()`.
    pub fn type_name(&self) -> &'static str {
        match self.kind {
            FlagKind::Bool => "bool",
            FlagKind::String => "string",
            FlagKind::BoundedInt { .. } => "int",
        }
    }
}

#[derive(Clone, Debug)]
pub struct Command {
    /// Cobra's `Use`: the name, then argument words.
    pub usage: &'static str,
    pub aliases: Vec<&'static str>,
    pub short: String,
    pub long: String,
    pub example: &'static str,
    pub args: Args,
    pub flags: Vec<Flag>,
    pub persistent: Vec<Flag>,
    pub run: Option<Run>,
    pub hidden: bool,
    /// The root's help command (Cobra hides it from suggestions).
    pub is_help: bool,
    pub version: Option<String>,
    /// Cobra's `DisableFlagsInUseLine`.
    pub usage_without_flags: bool,
    pub children: Vec<Command>,
}

impl Command {
    pub fn new(usage: &'static str, short: &str) -> Command {
        Command {
            usage,
            aliases: Vec::new(),
            short: short.to_string(),
            long: String::new(),
            example: "",
            args: Args::Legacy,
            flags: Vec::new(),
            persistent: Vec::new(),
            run: None,
            hidden: false,
            is_help: false,
            version: None,
            usage_without_flags: false,
            children: Vec::new(),
        }
    }

    pub fn aliases(mut self, aliases: &[&'static str]) -> Command {
        self.aliases = aliases.to_vec();
        self
    }

    pub fn long(mut self, long: &str) -> Command {
        self.long = long.to_string();
        self
    }

    pub fn example(mut self, example: &'static str) -> Command {
        self.example = example;
        self
    }

    pub fn args(mut self, args: Args) -> Command {
        self.args = args;
        self
    }

    pub fn exact(self, n: usize) -> Command {
        self.args(Args::Exact(n))
    }

    pub fn flag(mut self, flag: Flag) -> Command {
        self.flags.push(flag);
        self
    }

    pub fn run(mut self, run: Run) -> Command {
        self.run = Some(run);
        self
    }

    pub fn children(mut self, mut children: Vec<Command>) -> Command {
        self.children.append(&mut children);
        // Cobra sorts commands by name before listing them.
        self.children.sort_by(|a, b| a.name().cmp(b.name()));
        self
    }

    /// The first word of `usage`.
    pub fn name(&self) -> &'static str {
        self.usage.split(' ').next().unwrap_or(self.usage)
    }

    pub fn runnable(&self) -> bool {
        self.run.is_some()
    }

    pub fn is_available(&self) -> bool {
        !self.hidden && !self.is_help && (self.runnable() || self.has_available_children())
    }

    pub fn has_available_children(&self) -> bool {
        self.children.iter().any(Command::is_available)
    }

    /// The longest child name, for Cobra's name padding (minimum 11).
    pub fn name_padding(&self) -> usize {
        self.children
            .iter()
            .map(|c| c.name().len())
            .max()
            .unwrap_or(0)
            .max(11)
    }

    /// The child named `name` or with that alias (Cobra's `findNext`).
    pub fn child(&self, name: &str) -> Option<usize> {
        self.children
            .iter()
            .position(|c| c.name() == name || c.aliases.contains(&name))
    }
}

/// A path from the root to one command, as child indexes.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct CmdPath(pub Vec<usize>);

impl CmdPath {
    pub fn resolve<'a>(&self, root: &'a Command) -> &'a Command {
        self.0.iter().fold(root, |cmd, &i| &cmd.children[i])
    }

    /// The commands from the root down to this one.
    pub fn chain<'a>(&self, root: &'a Command) -> Vec<&'a Command> {
        let mut out = vec![root];
        let mut cmd = root;
        for &i in &self.0 {
            cmd = &cmd.children[i];
            out.push(cmd);
        }
        out
    }

    /// Cobra's `CommandPath`: the names joined with spaces.
    pub fn command_path(&self, root: &Command) -> String {
        self.chain(root)
            .iter()
            .map(|c| c.name())
            .collect::<Vec<_>>()
            .join(" ")
    }

    pub fn child(&self, index: usize) -> CmdPath {
        let mut path = self.0.clone();
        path.push(index);
        CmdPath(path)
    }

    pub fn is_root(&self) -> bool {
        self.0.is_empty()
    }
}
