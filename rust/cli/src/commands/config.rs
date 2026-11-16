//! config set-host, set-key, set-ca and show: local settings only.

use labtether_cli_client::{Secret, tls::validate_ca_file, validate_base_url};
use serde_json::Value;

use super::{Ctx, leaf};
use crate::cobra::{Args, Command};
use crate::errors::CliError;
use crate::output::{object, string};
use crate::paths::absolute;
use crate::settings::{load_config, save_config};

const MAX_KEY_BYTES: usize = 64 * 1024;

pub fn command() -> Command {
    Command::new("config", "Save the hub connection settings").children(vec![
        leaf("set-host <url>", "Set the hub URL", set_host).exact(1),
        leaf(
            "set-key",
            "Save an API key, read from a prompt or stdin",
            set_key,
        )
        .args(Args::None),
        leaf(
            "set-ca <pem-file>",
            "Trust a private CA certificate bundle for the hub",
            set_ca,
        )
        .exact(1),
        leaf("show", "Show the saved settings", show),
    ])
}

fn loaded(ctx: &Ctx) -> Result<crate::settings::Config, CliError> {
    match load_config(ctx.env) {
        (_, Some(err)) => Err(CliError::Config(err)),
        (config, None) => Ok(config),
    }
}

fn save(ctx: &Ctx, config: &crate::settings::Config) -> Result<(), CliError> {
    save_config(ctx.env, config).map_err(|e| CliError::Config(format!("save config: {e}")))
}

/// The Go CLI's `printStatusResult`.
fn status(ctx: &mut Ctx, json: Value, text: String) {
    if ctx.json {
        ctx.out.print_json(&json);
    } else {
        ctx.out.print(&text);
    }
}

fn set_host(ctx: &mut Ctx) -> Result<(), CliError> {
    let mut config = loaded(ctx)?;
    let host = ctx.args[0].trim().trim_end_matches('/').to_string();
    validate_base_url(&host).map_err(CliError::Config)?;
    config.host = host.clone();
    save(ctx, &config)?;
    status(
        ctx,
        object(&[("host", string(&host))]),
        format!("Hub host set to: {host}\n"),
    );
    Ok(())
}

/// The Go CLI's `readAPIKey`: a prompt without echo on a terminal, else
/// standard input up to 64 KiB.
fn read_api_key(ctx: &mut Ctx) -> Result<String, CliError> {
    let data = if ctx.stdin.is_terminal() {
        ctx.out.eprint("API key: ");
        let key = ctx.stdin.read_password();
        ctx.out.eprint("\n");
        key.map(String::into_bytes)
    } else {
        ctx.stdin.read_limited(MAX_KEY_BYTES as u64 + 1)
    };
    let data = data.map_err(|e| CliError::Config(format!("read API key: {e}")))?;
    if data.len() > MAX_KEY_BYTES {
        return Err(CliError::Config("API key exceeds 64 KiB".into()));
    }
    let key = String::from_utf8_lossy(&data).trim().to_string();
    if key.is_empty() {
        return Err(CliError::Config("API key is required".into()));
    }
    Ok(key)
}

fn set_key(ctx: &mut Ctx) -> Result<(), CliError> {
    let mut config = loaded(ctx)?;
    // Refused before the key is asked for: it could not be saved. (The Go
    // CLI's fix asks for the key first, then refuses.)
    if ctx.env.config_dir().is_none() {
        return Err(CliError::Config(format!(
            "save config: {}",
            crate::settings::no_home_folder()
        )));
    }
    config.api_key = Some(Secret::new(read_api_key(ctx)?));
    save(ctx, &config)?;
    status(
        ctx,
        object(&[("saved", Value::Bool(true))]),
        "API key saved.\n".into(),
    );
    Ok(())
}

fn set_ca(ctx: &mut Ctx) -> Result<(), CliError> {
    let mut config = loaded(ctx)?;
    let path = absolute(ctx.args[0].trim())
        .map_err(|e| CliError::Config(format!("resolve TLS CA file: {e}")))?;
    validate_ca_file(&path).map_err(|e| CliError::Config(e.to_string()))?;
    config.tls_ca_file = path.clone();
    save(ctx, &config)?;
    status(
        ctx,
        object(&[("tls_ca_file", string(&path))]),
        format!("TLS CA file set to: {path}\n"),
    );
    Ok(())
}

fn show(ctx: &mut Ctx) -> Result<(), CliError> {
    let Some(path) = ctx.env.config_path() else {
        return Err(CliError::Config(crate::settings::no_home_folder()));
    };
    let config = loaded(ctx)?;
    let key = if config
        .api_key
        .as_ref()
        .is_some_and(|k| !k.expose().is_empty())
    {
        "(set)"
    } else {
        "(not set)"
    };
    if ctx.json {
        ctx.out.print_json(&object(&[
            ("host", string(&config.host)),
            ("api_key", string(key)),
            ("tls_ca_file", string(&config.tls_ca_file)),
            ("config", string(&path)),
        ]));
        return Ok(());
    }
    let ca = if config.tls_ca_file.trim().is_empty() {
        "(system trust only)"
    } else {
        &config.tls_ca_file
    };
    ctx.out.print(&format!(
        "Host:    {}\nAPI key: {key}\nTLS CA:  {ca}\nConfig:  {path}\n",
        config.host
    ));
    Ok(())
}
