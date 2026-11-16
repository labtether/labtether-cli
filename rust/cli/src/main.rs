//! `labtether-cli-rs`: reads the arguments, runs one command, writes its
//! output and exits with its code.
#![forbid(unsafe_code)]

use std::io::Write;

use labtether_cli_rs::app::{Stdin, execute};
use labtether_cli_rs::output::Out;
use labtether_cli_rs::settings::Env;

fn main() {
    let args: Vec<String> = std::env::args_os()
        .skip(1)
        .map(|a| a.to_string_lossy().into_owned())
        .collect();
    let env = Env::from_process();
    let mut out = Out {
        sink: Some(Box::new(std::io::stdout())),
        ..Out::default()
    };
    let code = execute(&args, &env, &mut Stdin, &mut out, None);
    let _ = std::io::stdout().write_all(&out.stdout);
    let _ = std::io::stdout().flush();
    let _ = std::io::stderr().write_all(&out.stderr);
    std::process::exit(code);
}
