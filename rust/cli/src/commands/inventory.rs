//! devices, agents, hub, search, whoami, credentials and tls.

use serde_json::Value;

use crate::output::gofloat::{format_v, parse_f64};

use super::common::{check_any_map, decode_object, decode_rows, rows, typed};
use super::{Ctx, leaf, not_yet};
use crate::cobra::{Command, Flag};
use crate::errors::CliError;
use crate::output::cells::{Numbers, cell_text, clean_cell, status_word, text, time_text};
use crate::output::table::{col, render, tabwrite};
use crate::types::inventory::*;

pub fn commands() -> Vec<Command> {
    vec![
        Command::new("devices", "List devices and show their details")
            .aliases(&["assets"])
            .example("  labtether-cli devices list\n  labtether-cli devices list --online\n  labtether-cli devices get nas01")
            .children(vec![
                leaf("list", "List the devices this API key can reach", devices_list)
                    .flag(Flag::bool("online", "Show only online devices")),
                leaf("get <device>", "Show one device", asset_get).exact(1),
            ]),
        Command::new("agents", "List agents and approve new ones").children(vec![
            leaf("list", "List connected agents", agents_list),
            leaf("get <id>", "Show the device an agent runs on", asset_get).exact(1),
            leaf("pending", "List agents waiting for approval", agents_pending),
            not_yet("approve <id>", "Approve a pending agent").exact(1),
            not_yet("reject <id>", "Reject a pending agent").exact(1),
        ]),
        Command::new("hub", "Show hub status and connected agents").children(vec![
            leaf("status", "Show hub status", |ctx| ctx.get_and_print("/api/v2/hub/status")),
            leaf("agents", "List agents connected to the hub", hub_agents),
        ]),
        leaf("search <query>", "Search devices and groups", search).exact(1),
        leaf("whoami", "Show this API key's role, scopes and devices", whoami),
        Command::new("credentials", "Show saved credentials without their secrets").children(vec![
            leaf("list", "List saved credentials", credentials_list),
            leaf("get <id>", "Show one credential", |ctx| {
                let path = format!("/api/v2/credentials/profiles/{}", ctx.arg(0));
                ctx.get_and_print(&path)
            })
            .exact(1),
        ]),
        Command::new("tls", "Show the hub's TLS certificate")
            .children(vec![leaf("status", "Show TLS certificate status", |ctx| ctx.get_and_print("/api/v2/hub/tls"))]),
    ]
}

fn asset_get(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/assets/{}", ctx.arg(0));
    ctx.get_and_print(&path)
}

fn devices_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let mut path = "/api/v2/assets".to_string();
    if ctx.flags.bool("online") {
        path.push_str("?status=online");
    }
    let data = ctx.get(&path)?;
    if ctx.json {
        return ctx.print_data(&data);
    }
    let rows = decode_rows(&data)
        .map_err(|e| CliError::Failed(format!("failed to parse response: {e}")))?;
    let devices: Vec<Device> =
        typed(rows).map_err(|e| CliError::Failed(format!("failed to parse response: {e}")))?;
    ctx.out.print(&render(
        &devices,
        &[
            col("ID", |d: &Device| text(d.id.as_ref())),
            col("PLATFORM", |d: &Device| text(d.platform.as_ref())),
            col("STATUS", |d: &Device| {
                device_status(d.status.as_ref(), d.last_seen_at.as_ref())
            }),
            col("ADDRESS", device_address),
            col("NAME", |d: &Device| text(d.name.as_ref())),
        ],
    ));
    Ok(())
}

fn agents_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/agents")?;
    if ctx.json {
        return ctx.print_data(&data);
    }
    let mut agents: Vec<Agent> = rows(&data, "assetsInfo")?;
    agents.sort_by_key(|a| text(a.id.as_ref()));
    ctx.out.print(&render(
        &agents,
        &[
            col("ID", |a: &Agent| text(a.id.as_ref())),
            col("PLATFORM", |a: &Agent| text(a.platform.as_ref())),
        ],
    ));
    Ok(())
}

fn agents_pending(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/agents/pending")?;
    ctx.print_list(
        &data,
        "agents",
        &[
            col("ID", |a: &PendingAgent| text(a.asset_id.as_ref())),
            col("HOSTNAME", |a: &PendingAgent| text(a.hostname.as_ref())),
            col("PLATFORM", |a: &PendingAgent| text(a.platform.as_ref())),
            col("IP", |a: &PendingAgent| text(a.remote_ip.as_ref())),
            col("CONNECTED", |a: &PendingAgent| {
                time_text(a.connected_at.as_ref())
            }),
            col("VERIFIED", |a: &PendingAgent| {
                text(a.identity_verified.as_ref())
            }),
        ],
    )
}

fn hub_agents(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/hub/agents")?;
    ctx.print_list(
        &data,
        "presence",
        &[
            col("DEVICE", |p: &Presence| text(p.asset_id.as_ref())),
            col("TRANSPORT", |p: &Presence| text(p.transport.as_ref())),
            col("CONNECTED", |p: &Presence| {
                time_text(p.connected_at.as_ref())
            }),
            col("LAST HEARTBEAT", |p: &Presence| {
                time_text(p.last_heartbeat_at.as_ref())
            }),
        ],
    )
}

fn search(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/search?q={}", ctx.query_arg(0));
    let data = ctx.get(&path)?;
    if ctx.json {
        return ctx.print_data(&data);
    }
    decode_object(&data)?;
    let mut results = Vec::new();
    for field in ["assets", "groups"] {
        let Some(raw) = data.get(field) else { continue };
        let found = decode_rows(raw)
            .and_then(typed::<SearchResult>)
            .map_err(|e| CliError::Failed(format!("decode response data: {field}: {e}")))?;
        results.extend(found);
    }
    if results.is_empty() {
        ctx.out.print("No results.\n");
        return Ok(());
    }
    ctx.out.print(&render(
        &results,
        &[
            col("TYPE", |r: &SearchResult| result_kind(r.kind.as_ref())),
            col("ID", |r: &SearchResult| text(r.id.as_ref())),
            col("NAME", |r: &SearchResult| text(r.name.as_ref())),
            col("STATUS", |r: &SearchResult| {
                device_status(r.status.as_ref(), r.last_seen_at.as_ref())
            }),
            col("PLATFORM", |r: &SearchResult| text(r.platform.as_ref())),
        ],
    ));
    Ok(())
}

fn credentials_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/credentials/profiles")?;
    ctx.print_list(
        &data,
        "profiles",
        &[
            col("ID", |c: &CredentialProfile| text(c.id.as_ref())),
            col("NAME", |c: &CredentialProfile| text(c.name.as_ref())),
            col("KIND", |c: &CredentialProfile| text(c.kind.as_ref())),
            col("USERNAME", |c: &CredentialProfile| {
                text(c.username.as_ref())
            }),
            col("STATUS", |c: &CredentialProfile| {
                status_word(c.status.as_ref())
            }),
        ],
    )
}

/// Go's `fmt.Print` of a value decoded into `any`.
fn go_sprint(value: &Value) -> String {
    match value {
        Value::Null => "<nil>".to_string(),
        Value::Bool(b) => b.to_string(),
        Value::String(s) => s.clone(),
        Value::Number(n) => parse_f64(&n.to_string())
            .map(format_v)
            .unwrap_or_else(|| n.to_string()),
        Value::Array(items) => format!(
            "[{}]",
            items.iter().map(go_sprint).collect::<Vec<_>>().join(" ")
        ),
        Value::Object(map) => {
            let parts: Vec<String> = map
                .iter()
                .map(|(k, v)| format!("{k}:{}", go_sprint(v)))
                .collect();
            format!("map[{}]", parts.join(" "))
        }
    }
}

fn whoami(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/whoami")?;
    if ctx.json {
        return ctx.print_data(&data);
    }
    let parse_failed = |e: String| CliError::Failed(format!("failed to parse response: {e}"));
    check_any_map(&data).map_err(parse_failed)?;
    if let Some(number) = out_of_range(&data) {
        return Err(parse_failed(format!("the number {number} is too large")));
    }
    let cell = |key: &str| clean_cell(&cell_text(data.get(key), Numbers::Float));
    let mut out = format!(
        "Auth:     {}\nRole:     {}\n",
        cell("auth_type"),
        cell("role")
    );
    if data.get("key_name").is_some() {
        out.push_str(&format!("Key:      {}\n", cell("key_name")));
    }
    if let Some(Value::Array(scopes)) = data.get("scopes") {
        let scopes: Vec<String> = scopes.iter().map(go_sprint).collect();
        out.push_str(&format!("Scopes:   {}\n", scopes.join(", ")));
    }
    if let Some(Value::Array(devices)) = data.get("available_assets") {
        out.push_str(&format!("Devices:  {}\n", devices.len()));
        let lines: Vec<Vec<String>> = devices
            .iter()
            .filter(|d| matches!(d, Value::Object(_)))
            .map(|d| {
                let field = |k: &str| clean_cell(&cell_text(d.get(k), Numbers::Float));
                let status = if d.get("online") == Some(&Value::Bool(true)) {
                    "Online"
                } else {
                    "Offline"
                };
                vec![
                    format!("  {}", field("id")),
                    field("platform"),
                    status.to_string(),
                ]
            })
            .collect();
        out.push_str(&tabwrite(&lines));
    }
    ctx.out.print(&out);
    Ok(())
}

/// The first number Go's float64 cannot hold, which makes the Go CLI's
/// `whoami` fail to read the answer.
fn out_of_range(value: &Value) -> Option<String> {
    match value {
        Value::Number(n) => parse_f64(&n.to_string()).is_none().then(|| n.to_string()),
        Value::Array(items) => items.iter().find_map(out_of_range),
        Value::Object(map) => map.values().find_map(out_of_range),
        _ => None,
    }
}
