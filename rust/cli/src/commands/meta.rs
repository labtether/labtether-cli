//! Cobra's help and completion commands.

use super::{Ctx, complete, leaf};
use crate::cobra::{Args, Command, Flag, find, help};
use crate::errors::CliError;

pub fn commands() -> Vec<Command> {
    let mut help_command = leaf("help [command]", "Help about any command", help_run).long(
        "Help provides help for any command in the application.\n\
         Simply type labtether-cli help [path to command] for full details.",
    );
    help_command.is_help = true;
    // Cobra's scripts, and the hidden `__complete` command they call
    // (`complete.rs`).
    let shell = |usage, short, long: &str, run: super::Handler| {
        leaf(usage, short, run)
            .long(long)
            .args(Args::None)
            .flag(Flag::bool(
                "no-descriptions",
                "disable completion descriptions",
            ))
    };
    let mut bash = shell(
        "bash",
        "Generate the autocompletion script for bash",
        BASH_LONG,
        complete::bash,
    );
    bash.usage_without_flags = true;
    vec![
        help_command,
        Command::new(
            "completion",
            "Generate the autocompletion script for the specified shell",
        )
        .long(
            "Generate the autocompletion script for labtether-cli for the specified shell.\n\
                 See each sub-command's help for details on how to use the generated script.\n",
        )
        .args(Args::None)
        .children(vec![
            bash,
            shell(
                "zsh",
                "Generate the autocompletion script for zsh",
                ZSH_LONG,
                complete::zsh,
            ),
            shell(
                "fish",
                "Generate the autocompletion script for fish",
                FISH_LONG,
                complete::fish,
            ),
            shell(
                "powershell",
                "Generate the autocompletion script for powershell",
                POWERSHELL_LONG,
                complete::powershell,
            ),
        ]),
    ]
}

/// Go's `%#q` for a list of words: "[`a` `b`]".
fn backquoted(words: &[String]) -> String {
    let quoted: Vec<String> = words
        .iter()
        .map(|w| {
            if w.contains('`') || w.chars().any(|c| c.is_control() && c != '\t') {
                crate::cobra::quote::quote(w)
            } else {
                format!("`{w}`")
            }
        })
        .collect();
    format!("[{}]", quoted.join(" "))
}

/// Cobra's help command: the named command's help on stdout, or "Unknown
/// help topic" and the root's usage on stderr.
fn help_run(ctx: &mut Ctx) -> Result<(), CliError> {
    let (path, _, err) = find::find(ctx.root, &ctx.args);
    if err.is_some() {
        ctx.out
            .eprint(&format!("Unknown help topic {}\n", backquoted(&ctx.args)));
        ctx.out
            .eprint(&help::usage(ctx.root, &Default::default(), false));
        return Ok(());
    }
    ctx.out.print(&help::help(ctx.root, &path));
    Ok(())
}

const BASH_LONG: &str = "Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

\tsource <(labtether-cli completion bash)

To load completions for every new session, execute once:

#### Linux:

\tlabtether-cli completion bash > /etc/bash_completion.d/labtether-cli

#### macOS:

\tlabtether-cli completion bash > $(brew --prefix)/etc/bash_completion.d/labtether-cli

You will need to start a new shell for this setup to take effect.
";

const ZSH_LONG: &str = "Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

\techo \"autoload -U compinit; compinit\" >> ~/.zshrc

To load completions in your current shell session:

\tsource <(labtether-cli completion zsh)

To load completions for every new session, execute once:

#### Linux:

\tlabtether-cli completion zsh > \"${fpath[1]}/_labtether-cli\"

#### macOS:

\tlabtether-cli completion zsh > $(brew --prefix)/share/zsh/site-functions/_labtether-cli

You will need to start a new shell for this setup to take effect.
";

const FISH_LONG: &str = "Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

\tlabtether-cli completion fish | source

To load completions for every new session, execute once:

\tlabtether-cli completion fish > ~/.config/fish/completions/labtether-cli.fish

You will need to start a new shell for this setup to take effect.
";

const POWERSHELL_LONG: &str = "Generate the autocompletion script for powershell.

To load completions in your current shell session:

\tlabtether-cli completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.
";
