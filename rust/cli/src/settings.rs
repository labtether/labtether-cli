//! Settings: the config file, environment variables and flags, with the Go
//! CLI's precedence (flag > environment > config file), checks and
//! messages.

use std::io::Read;
use std::path::Path;

use labtether_cli_client::error::path_error;
use labtether_cli_client::{Client, ProxyEnv, Secret, validate_base_url};
use labtether_cli_platform::{
    create_private_file, harden_config_file, permissions_too_broad, replace_config_file,
};
use serde::Deserialize;
use serde_json::Value;

use crate::errors::CliError;
use crate::output::gojson::{Numbers, encode};
use crate::paths::join;

const MAX_CONFIG_BYTES: u64 = 1024 * 1024;
const MAX_KEY_BYTES: u64 = 64 * 1024;

/// The process environment, read once.
#[derive(Clone, Debug, Default)]
pub struct Env {
    pub host: Option<String>,
    pub api_key: Option<Secret>,
    pub ca_file: Option<String>,
    pub home: Option<String>,
    /// `LABTETHER_CLI_COMPLETION_DESCRIPTIONS`, else
    /// `COBRA_COMPLETION_DESCRIPTIONS` (Cobra reads both).
    pub completion_descriptions: Option<String>,
    /// `BASH_COMP_DEBUG_FILE`: where Cobra's completion appends its notes.
    pub comp_debug_file: Option<String>,
    /// HTTPS_PROXY, HTTP_PROXY and NO_PROXY, as Go's transport reads them.
    pub proxy: ProxyEnv,
}

impl Env {
    pub fn from_process() -> Env {
        let var = |name: &str| std::env::var(name).ok().filter(|v| !v.is_empty());
        Env {
            host: var("LABTETHER_HOST"),
            api_key: var("LABTETHER_API_KEY").map(Secret::new),
            ca_file: var("LABTETHER_TLS_CA_FILE"),
            home: if cfg!(windows) {
                var("USERPROFILE")
            } else {
                var("HOME")
            },
            completion_descriptions: var("LABTETHER_CLI_COMPLETION_DESCRIPTIONS")
                .or_else(|| var("COBRA_COMPLETION_DESCRIPTIONS")),
            comp_debug_file: var("BASH_COMP_DEBUG_FILE"),
            proxy: ProxyEnv::from_process(),
        }
    }

    /// `$HOME/.config/labtether`, or none without a home folder. There is
    /// no fallback: a shared folder such as the system temporary folder
    /// could hold a config file (and a Hub address) another user put there.
    /// The Go CLI used `<temp folder>/labtether`.
    pub fn config_dir(&self) -> Option<String> {
        self.home
            .as_ref()
            .map(|home| join(&[home, ".config", "labtether"]))
    }

    pub fn config_path(&self) -> Option<String> {
        self.config_dir().map(|dir| join(&[&dir, "config.json"]))
    }
}

/// Why settings cannot be read or saved without a home folder.
pub fn no_home_folder() -> String {
    let name = if cfg!(windows) { "USERPROFILE" } else { "HOME" };
    format!(
        "no home folder to keep settings in; set {name}, or set LABTETHER_HOST and LABTETHER_API_KEY"
    )
}

/// The saved settings.
#[derive(Clone, Debug, Default)]
pub struct Config {
    pub host: String,
    pub api_key: Option<Secret>,
    pub tls_ca_file: String,
}

/// The config file's fields. Unknown fields are ignored, as in Go.
#[derive(Deserialize)]
struct ConfigFile {
    #[serde(default)]
    host: Option<String>,
    #[serde(default)]
    api_key: Option<String>,
    #[serde(default)]
    tls_ca_file: Option<String>,
}

/// Decodes the config file. A file that does not parse, or a field of the
/// wrong type, gives a plain message (the Go CLI prints Go's decoder text,
/// and keeps the fields that did decode).
fn decode_config(data: &[u8]) -> (Config, Option<String>) {
    let parsed = serde_json::from_slice::<Value>(data).and_then(|top| match top {
        Value::Null => Ok(None),
        // serde would also read a list into the struct; Go accepts only an object.
        Value::Object(_) => ConfigFile::deserialize(&top).map(Some),
        other => Err(serde::de::Error::custom(format!(
            "it holds {}, not an object",
            crate::commands::common::kind(&other)
        ))),
    });
    match parsed {
        Ok(None) => (Config::default(), None),
        Ok(Some(file)) => (
            Config {
                host: file.host.unwrap_or_default(),
                api_key: file.api_key.map(Secret::new),
                tls_ca_file: file.tls_ca_file.unwrap_or_default(),
            },
            None,
        ),
        Err(err) => (
            Config::default(),
            Some(format!("the file is not a valid config ({err})")),
        ),
    }
}

/// Go's `loadConfig`: a missing file is an empty config; a symlink or
/// device is refused; the file is set to mode 0600 before it is read.
pub fn load_config(env: &Env) -> (Config, Option<String>) {
    let Some(path) = env.config_path() else {
        return (Config::default(), None);
    };
    let meta = match std::fs::symlink_metadata(&path) {
        Ok(meta) => meta,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return (Config::default(), None),
        Err(err) => {
            return (
                Config::default(),
                Some(format!(
                    "inspect config {path}: {}",
                    path_error("lstat", &path, &err)
                )),
            );
        }
    };
    if !meta.is_file() {
        return (
            Config::default(),
            Some(format!(
                "config {path} must be a regular file, not a symlink or device"
            )),
        );
    }
    if let Err(err) = harden_config_file(Path::new(&path)) {
        return (
            Config::default(),
            Some(format!(
                "protect config {path}: {}",
                path_error("chmod", &path, &err)
            )),
        );
    }
    let mut data = Vec::new();
    let read = std::fs::File::open(&path)
        .map_err(|e| format!("read config {path}: {}", path_error("open", &path, &e)))
        .and_then(|file| {
            file.take(MAX_CONFIG_BYTES + 1)
                .read_to_end(&mut data)
                .map_err(|e| format!("read config {path}: {}", path_error("read", &path, &e)))
        });
    if let Err(err) = read {
        return (Config::default(), Some(err));
    }
    if data.len() as u64 > MAX_CONFIG_BYTES {
        return (
            Config::default(),
            Some(format!("config {path} exceeds 1 MiB")),
        );
    }
    let (config, err) = decode_config(&data);
    (config, err.map(|e| format!("parse config {path}: {e}")))
}

fn push_json_string(out: &mut String, text: &str) {
    out.push_str(&encode(
        &Value::String(text.to_string()),
        Numbers::Text,
        false,
    ));
}

/// Go's `json.MarshalIndent(cfg, "", "  ")` for the config struct (fields
/// in struct order, so not through a sorted map).
fn config_json(config: &Config) -> String {
    let mut out = String::from("{\n  \"host\": ");
    push_json_string(&mut out, &config.host);
    out.push_str(",\n  \"api_key\": ");
    push_json_string(
        &mut out,
        config
            .api_key
            .as_ref()
            .map(Secret::expose)
            .unwrap_or_default(),
    );
    if !config.tls_ca_file.is_empty() {
        out.push_str(",\n  \"tls_ca_file\": ");
        push_json_string(&mut out, &config.tls_ca_file);
    }
    out.push_str("\n}");
    out
}

/// Go's `saveConfig`: a protected temporary file renamed over the config.
pub fn save_config(env: &Env, config: &Config) -> Result<(), String> {
    let (Some(dir), Some(path)) = (env.config_dir(), env.config_path()) else {
        return Err(no_home_folder());
    };
    create_private_dir(&dir).map_err(|e| path_error("mkdir", &dir, &e))?;
    let nonce = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .subsec_nanos();
    let tmp = join(&[&dir, &format!(".config-{}{nonce}.tmp", std::process::id())]);
    let result = (|| {
        // Private from the moment it exists (mode 0600; on Windows the
        // private ACL at creation and no sharing): the key is written next.
        let mut file =
            create_private_file(Path::new(&tmp)).map_err(|e| path_error("open", &tmp, &e))?;
        std::io::Write::write_all(&mut file, config_json(config).as_bytes())
            .map_err(|e| path_error("write", &tmp, &e))?;
        file.sync_all().map_err(|e| path_error("sync", &tmp, &e))?;
        drop(file);
        replace_config_file(Path::new(&tmp), Path::new(&path)).map_err(|e| {
            format!(
                "rename {tmp} {path}: {}",
                labtether_cli_client::error::errno_text(&e)
            )
        })?;
        harden_config_file(Path::new(&path)).map_err(|e| path_error("chmod", &path, &e))
    })();
    let _ = std::fs::remove_file(&tmp);
    result
}

fn create_private_dir(dir: &str) -> std::io::Result<()> {
    let mut builder = std::fs::DirBuilder::new();
    builder.recursive(true);
    #[cfg(unix)]
    std::os::unix::fs::DirBuilderExt::mode(&mut builder, 0o700);
    builder.create(dir)
}

/// Go's `readProtectedSecretFile` for `--api-key-file`.
pub fn read_key_file(path: &str) -> Result<Secret, String> {
    let mut file = std::fs::File::open(path)
        .map_err(|e| format!("read API key file: {}", path_error("open", path, &e)))?;
    let meta = file
        .metadata()
        .map_err(|e| format!("inspect API key file: {}", path_error("stat", path, &e)))?;
    if !meta.is_file() {
        return Err("API key file must be a regular file".to_string());
    }
    if permissions_too_broad(&meta) {
        return Err("API key file permissions are too broad; require mode 0600".to_string());
    }
    let mut data = Vec::new();
    file.by_ref()
        .take(MAX_KEY_BYTES + 1)
        .read_to_end(&mut data)
        .map_err(|e| format!("read API key file: {}", path_error("read", path, &e)))?;
    if data.len() as u64 > MAX_KEY_BYTES {
        return Err("API key file exceeds 64 KiB".to_string());
    }
    let secret = String::from_utf8_lossy(&data).trim().to_string();
    if secret.is_empty() {
        return Err("API key file is empty".to_string());
    }
    Ok(Secret::new(secret))
}

/// The connection flags given on the command line.
#[derive(Clone, Debug, Default)]
pub struct Flags {
    pub host: String,
    pub api_key: String,
    pub api_key_file: String,
    pub ca_file: String,
}

/// The settings a Hub command uses, after precedence and checks.
#[derive(Debug)]
pub struct Resolved {
    pub host: String,
    pub api_key: Secret,
    pub ca_file: String,
}

/// Go's `clientFromSettings`, in the same order, so the first problem
/// reported is the same. Every error is a settings error (exit 4).
pub fn resolve(env: &Env, flags: &Flags) -> Result<Resolved, CliError> {
    let config_error = |m: String| CliError::Config(m);
    let (config, config_err) = load_config(env);
    let mut host = config.host;
    let mut key = config
        .api_key
        .map(|k| k.expose().to_string())
        .unwrap_or_default();
    let mut ca_file = config.tls_ca_file;
    if let Some(v) = &env.host {
        host = v.clone();
    }
    if let Some(v) = &env.api_key {
        key = v.expose().to_string();
    }
    if let Some(v) = &env.ca_file {
        ca_file = v.clone();
    }
    if !flags.api_key_file.is_empty() {
        key = read_key_file(&flags.api_key_file)
            .map_err(config_error)?
            .expose()
            .to_string();
    }
    if !flags.host.is_empty() {
        host = flags.host.clone();
    }
    if !flags.ca_file.is_empty() {
        ca_file = flags.ca_file.clone();
    }
    if !flags.api_key.is_empty() {
        return Err(config_error(
            "refusing API key in command-line arguments; use --api-key-file or LABTETHER_API_KEY"
                .to_string(),
        ));
    }
    let (host, key, ca_file) = (
        host.trim().to_string(),
        key.trim().to_string(),
        ca_file.trim().to_string(),
    );
    if let Some(err) = config_err
        && (host.is_empty() || key.is_empty())
    {
        return Err(config_error(err));
    }
    if host.is_empty() {
        return Err(config_error(
            "hub host not configured; run: labtether-cli config set-host <url>".to_string(),
        ));
    }
    if key.is_empty() {
        return Err(config_error(
            "API key not configured; run: labtether-cli config set-key".to_string(),
        ));
    }
    validate_base_url(&host).map_err(config_error)?;
    Ok(Resolved {
        host,
        api_key: Secret::new(key),
        ca_file,
    })
}

/// Go's `newClient`.
pub fn new_client(env: &Env, flags: &Flags) -> Result<Client, CliError> {
    let resolved = resolve(env, flags)?;
    let ca = (!resolved.ca_file.is_empty()).then_some(resolved.ca_file.as_str());
    Client::new(&resolved.host, resolved.api_key, ca, env.proxy.clone())
        .map_err(|e| CliError::Config(e.to_string()))
}

#[cfg(test)]
#[path = "settings_tests.rs"]
mod tests;
