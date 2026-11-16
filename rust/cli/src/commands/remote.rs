//! Docker and Proxmox power actions and `connectors test`: commands that
//! read a Hub answer before or after the change to decide what to say.

use labtether_cli_client::Method;
use serde_json::{Map, Value};

use super::Ctx;
use super::actions::LONG_ACTION;
use super::common::{as_object, equal_fold, go_lower, object_member, str_member};
use crate::cobra::quote::quote;
use crate::errors::CliError;
use crate::output::gofloat::parse_f64;
use crate::output::gojson::{Numbers, encode};

fn failed(prefix: &str) -> impl Fn(String) -> CliError + '_ {
    move |e| CliError::Failed(format!("{prefix}: {e}"))
}

/// `{"status": ..., "message": ...}` from `map`.
fn outcome(map: Option<&Map<String, Value>>) -> Result<(String, String), String> {
    Ok((str_member(map, "status")?, str_member(map, "message")?))
}

fn docker_action(ctx: &mut Ctx, verb: &str, done: &str) -> Result<(), CliError> {
    let path = format!("/api/v2/docker/containers/{}/action", ctx.arg(0));
    let body = Ctx::body(&[("action", Value::String(format!("container.{verb}")))]);
    let data = ctx.send(Method::Post, &path, Some(body))?;
    let (status, message) = as_object(Some(&data))
        .and_then(|top| object_member(top, "result"))
        .and_then(outcome)
        .map_err(failed("decode response data"))?;
    if status != "succeeded" {
        return Err(CliError::Failed(if message.is_empty() {
            format!("Docker action failed with status {}", quote(&status))
        } else {
            format!("Docker action failed: {message}")
        }));
    }
    let said = format!("Container {} {done}\n", ctx.args[0]);
    ctx.print_status(&data, &said)
}

pub fn docker_start(ctx: &mut Ctx) -> Result<(), CliError> {
    docker_action(ctx, "start", "started")
}

pub fn docker_stop(ctx: &mut Ctx) -> Result<(), CliError> {
    docker_action(ctx, "stop", "stopped")
}

pub fn docker_restart(ctx: &mut Ctx) -> Result<(), CliError> {
    docker_action(ctx, "restart", "restarted")
}

/// The Go CLI's `requestProxmoxPowerAction`: checks that the device is a
/// Proxmox VM or container, then runs the connector action on it.
fn proxmox_action(ctx: &mut Ctx, verb: &str, done: &str) -> Result<(), CliError> {
    let client = ctx.client_with(LONG_ACTION)?;
    let id = ctx.args[0].clone();
    let details = client.get(&format!("/api/v2/assets/{}", ctx.arg(0)))?;
    let asset = as_object(Some(&details))
        .and_then(|top| object_member(top, "asset"))
        .map_err(failed("decode device details"))?;
    let source = str_member(asset, "source").map_err(failed("decode device details"))?;
    let kind = str_member(asset, "type").map_err(failed("decode device details"))?;
    if !equal_fold(&source, "proxmox") {
        return Err(CliError::Failed(format!(
            "device {} is not from Proxmox",
            quote(&id)
        )));
    }
    let prefix = match go_lower(&kind).as_str() {
        "vm" => "vm",
        "container" => "ct",
        _ => {
            return Err(CliError::Failed(format!(
                "device {} is not a Proxmox VM or container",
                quote(&id)
            )));
        }
    };
    let path = format!("/api/v2/connectors/proxmox/actions/{prefix}.{verb}/execute");
    let body = Ctx::body(&[("target_id", Value::String(id.clone()))]);
    let result = client.send(Method::Post, &path, Some(&body))?;
    let (status, message) = as_object(Some(&result))
        .and_then(outcome)
        .map_err(failed("decode Proxmox action result"))?;
    if !equal_fold(&status, "succeeded") {
        return Err(CliError::Failed(if message.is_empty() {
            format!("Proxmox action failed with status {}", quote(&status))
        } else {
            format!("Proxmox action failed: {message}")
        }));
    }
    ctx.print_status(&result, &format!("Proxmox guest {id} {done}\n"))
}

pub fn proxmox_start(ctx: &mut Ctx) -> Result<(), CliError> {
    proxmox_action(ctx, "start", "started")
}

pub fn proxmox_stop(ctx: &mut Ctx) -> Result<(), CliError> {
    proxmox_action(ctx, "stop", "stopped")
}

pub fn proxmox_restart(ctx: &mut Ctx) -> Result<(), CliError> {
    proxmox_action(ctx, "reboot", "restarted")
}

/// The settings `connectors test` sends from a saved connection; its
/// secret stays in the Hub under `credential_id`.
const TEST_SETTINGS: [&str; 7] = [
    "base_url",
    "credential_id",
    "auth_method",
    "token_id",
    "username",
    "skip_verify",
    "ca_pem",
];

/// Whether a value holds a number Go's float64 cannot hold (Go's decoder
/// fails on it).
fn beyond_float(value: &Value) -> bool {
    match value {
        Value::Number(n) => parse_f64(&n.to_string()).is_none(),
        Value::Array(items) => items.iter().any(beyond_float),
        Value::Object(map) => map.values().any(beyond_float),
        _ => false,
    }
}

struct Saved {
    id: String,
    kind: String,
    config: Map<String, Value>,
}

/// The Go CLI's decode of `{"collectors": [{id, collector_type, config}]}`.
fn saved_connections(data: &Value) -> Result<Vec<Saved>, String> {
    let top = as_object(Some(data))?;
    let list = match top.and_then(|t| t.get("collectors")) {
        None | Some(Value::Null) => return Ok(Vec::new()),
        Some(Value::Array(items)) => items,
        Some(other) => {
            return Err(format!(
                "collectors is {}, not a list",
                super::common::kind(other)
            ));
        }
    };
    let mut out = Vec::new();
    for (i, item) in list.iter().enumerate() {
        let item = as_object(Some(item)).map_err(|e| format!("item {i}: {e}"))?;
        let config = object_member(item, "config").map_err(|e| format!("item {i}: {e}"))?;
        if config.is_some_and(|c| c.values().any(beyond_float)) {
            return Err(format!(
                "item {i}: a setting holds a number too large to read"
            ));
        }
        out.push(Saved {
            id: str_member(item, "id").map_err(|e| format!("item {i}: {e}"))?,
            kind: str_member(item, "collector_type").map_err(|e| format!("item {i}: {e}"))?,
            config: config.cloned().unwrap_or_default(),
        });
    }
    Ok(out)
}

/// Collectors name Home Assistant without the hyphen the connector list
/// uses.
fn collector_type_for(connector: &str) -> &str {
    if connector == "home-assistant" {
        "homeassistant"
    } else {
        connector
    }
}

/// The Go CLI's `connectorTestRequest`: the connector to test and the
/// settings of the saved connection named by a collector ID, or by a
/// connector with exactly one saved connection.
fn test_request(ctx: &mut Ctx, target: &str) -> Result<(String, Vec<u8>), CliError> {
    // The Docker test checks the Hub's own Docker connection.
    if target == "docker" {
        return Ok((target.to_string(), b"{}".to_vec()));
    }
    let data = ctx.get("/api/v2/collectors")?;
    let saved = saved_connections(&data).map_err(failed("decode saved connections"))?;
    let mut matches: Vec<&Saved> = Vec::new();
    for connection in &saved {
        if connection.id == target {
            matches = vec![connection];
            break;
        }
        if connection.kind == collector_type_for(target) {
            matches.push(connection);
        }
    }
    let chosen = match matches.as_slice() {
        [] => {
            return Err(CliError::Failed(format!(
                "no saved {target} connection to test"
            )));
        }
        [one] => one,
        many => {
            let ids: Vec<&str> = many.iter().map(|s| s.id.as_str()).collect();
            return Err(CliError::Failed(format!(
                "{} saved {target} connections; test one by its ID: {}",
                many.len(),
                ids.join(", ")
            )));
        }
    };
    let body: Map<String, Value> = TEST_SETTINGS
        .iter()
        .filter_map(|key| {
            chosen
                .config
                .get(*key)
                .map(|v| (key.to_string(), v.clone()))
        })
        .collect();
    let connector = if chosen.kind == "homeassistant" {
        "home-assistant".to_string()
    } else {
        chosen.kind.clone()
    };
    // Go re-encodes the settings it decoded into `any`: numbers in its
    // float64 form.
    let text = encode(&Value::Object(body), Numbers::Float, false);
    Ok((connector, text.into_bytes()))
}

/// `connectors test` prints the Hub's data as JSON.
pub fn connectors_test(ctx: &mut Ctx) -> Result<(), CliError> {
    ctx.client()?;
    let target = ctx.args[0].clone();
    let (connector, body) = test_request(ctx, &target)?;
    let path = format!(
        "/api/v2/connectors/{}/test",
        labtether_cli_client::escape::path_escape(&connector)
    );
    let data = ctx.send(Method::Post, &path, Some(body))?;
    ctx.print_data(&data)
}
