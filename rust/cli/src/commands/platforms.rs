//! proxmox, truenas, pbs and portainer.

use serde::Deserialize;
use serde_json::Value;

use super::common::{decode_list, decode_rows, kind, typed};
use super::{Ctx, leaf, not_yet};
use crate::cobra::Command;
use crate::errors::CliError;
use crate::output::cells::{byte_size, short_id, status_word, text, time_text, unix_time_text};
use crate::output::table::{col, render};
use crate::types::platforms::*;

fn asset_leaf(usage: &'static str, short: &str, handler: super::Handler) -> Command {
    leaf(usage, short, handler).exact(1)
}

fn asset_route(ctx: &mut Ctx, area: &str, suffix: &str) -> Result<(), CliError> {
    let path = format!("/api/v2/{area}/assets/{}/{suffix}", ctx.arg(0));
    ctx.get_and_print(&path)
}

pub fn commands() -> Vec<Command> {
    vec![
        Command::new("proxmox", "Show and control Proxmox VMs and containers").children(vec![
            leaf("cluster-status", "Show Proxmox cluster status", |ctx| {
                ctx.get_and_print("/api/v2/proxmox/cluster/status")
            }),
            leaf("resources", "List Proxmox resources", proxmox_resources),
            leaf("nodes", "List Proxmox nodes", proxmox_nodes),
            asset_leaf("get <vm>", "Show one VM or container", |ctx| {
                asset_route(ctx, "proxmox", "details")
            }),
            not_yet("start <vm>", "Start a VM or container").exact(1),
            not_yet("stop <vm>", "Stop a VM or container").exact(1),
            not_yet("restart <vm>", "Restart a VM or container").exact(1),
            leaf("ceph-status", "Show Ceph status", |ctx| {
                ctx.get_and_print("/api/v2/proxmox/ceph/status")
            }),
        ]),
        Command::new("truenas", "Show TrueNAS pools, datasets and shares").children(vec![
            asset_leaf("get <device>", "Show one TrueNAS system", |ctx| {
                asset_route(ctx, "truenas", "overview")
            }),
            asset_leaf("pools <device>", "List pools on a TrueNAS system", |ctx| {
                asset_route(ctx, "truenas", "pools")
            }),
            asset_leaf(
                "datasets <device>",
                "List datasets on a TrueNAS system",
                |ctx| asset_route(ctx, "truenas", "datasets"),
            ),
            asset_leaf(
                "shares <device>",
                "List shares on a TrueNAS system",
                |ctx| asset_route(ctx, "truenas", "shares"),
            ),
        ]),
        Command::new("pbs", "Show Proxmox Backup Server datastores and snapshots").children(vec![
            asset_leaf("get <device>", "Show one PBS server", |ctx| {
                asset_route(ctx, "pbs", "details")
            }),
            asset_leaf(
                "datastores <device>",
                "List datastores on a PBS server",
                pbs_datastores,
            ),
            asset_leaf(
                "snapshots <device>",
                "List snapshots on a PBS server",
                pbs_snapshots,
            ),
        ]),
        Command::new("portainer", "Show Portainer stacks and containers").children(vec![
            asset_leaf("get <device>", "Show one Portainer environment", |ctx| {
                asset_route(ctx, "portainer", "overview")
            }),
            asset_leaf(
                "stacks <device>",
                "List stacks in a Portainer environment",
                portainer_stacks,
            ),
            asset_leaf(
                "containers <device>",
                "List containers in a Portainer environment",
                portainer_containers,
            ),
        ]),
    ]
}

fn proxmox_resources(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/proxmox/cluster/resources")?;
    ctx.print_list(
        &data,
        "resources",
        &[
            col("ID", |r: &ProxmoxResource| text(r.id.as_ref())),
            col("TYPE", |r: &ProxmoxResource| text(r.kind.as_ref())),
            col("NODE", |r: &ProxmoxResource| text(r.node.as_ref())),
            col("NAME", |r: &ProxmoxResource| text(r.name.as_ref())),
            col("STATUS", |r: &ProxmoxResource| {
                status_word(r.status.as_ref())
            }),
        ],
    )
}

fn is_node(row: &Value) -> bool {
    row.get("type") == Some(&Value::String("node".into()))
}

/// The Go CLI's decode into `struct { Resources []map[string]any }`.
#[derive(Deserialize)]
struct ResourceList {
    #[serde(default)]
    resources: Option<Vec<Option<serde_json::Map<String, Value>>>>,
}

fn decode_resources(data: &Value) -> Result<Vec<Value>, String> {
    match data {
        Value::Null => return Ok(Vec::new()),
        Value::Object(_) => {}
        other => return Err(format!("the hub sent {}, not an object", kind(other))),
    }
    let list = ResourceList::deserialize(data)
        .map_err(|e| format!("the hub's answer has an unexpected shape: {e}"))?;
    Ok(list
        .resources
        .unwrap_or_default()
        .into_iter()
        .flatten()
        .map(Value::Object)
        .collect())
}

fn proxmox_nodes(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/proxmox/cluster/resources")?;
    if ctx.json {
        let resources = decode_resources(&data)
            .map_err(|e| CliError::Failed(format!("decode Proxmox resources: {e}")))?;
        let nodes: Vec<Value> = resources.into_iter().filter(is_node).collect();
        return ctx.print_data(&Value::Array(nodes));
    }
    let rows: Vec<&Value> = decode_list(&data, "resources")?
        .into_iter()
        .filter(|r| is_node(r))
        .collect();
    let nodes: Vec<ProxmoxResource> = typed(rows)
        .map_err(|e| CliError::Failed(format!("decode response data: resources: {e}")))?;
    ctx.out.print(&render(
        &nodes,
        &[
            col("NODE", |r: &ProxmoxResource| text(r.node.as_ref())),
            col("STATUS", |r: &ProxmoxResource| {
                status_word(r.status.as_ref())
            }),
            col("CPUS", |r: &ProxmoxResource| text(r.maxcpu.as_ref())),
            col("MEMORY", |r: &ProxmoxResource| byte_size(r.maxmem.as_ref())),
        ],
    ));
    Ok(())
}

/// The PBS details fields the Go CLI's `pbsDatastoreList` reads.
#[derive(Deserialize)]
struct Details {
    #[serde(default)]
    datastore: Option<Value>,
    #[serde(default)]
    datastores: Option<Value>,
}

/// The Go CLI's `pbsDatastoreList`: the `datastores` list, or the one
/// `datastore` summary as a list, or an empty list.
fn datastore_list(data: &Value) -> Result<Value, CliError> {
    let details = match data {
        Value::Null => Details {
            datastore: None,
            datastores: None,
        },
        Value::Object(_) => Details::deserialize(data)
            .map_err(|e| CliError::Failed(format!("decode response data: {e}")))?,
        other => {
            return Err(CliError::Failed(format!(
                "decode response data: the hub sent {}, not an object",
                kind(other)
            )));
        }
    };
    Ok(match (details.datastores, details.datastore) {
        (Some(list), _) => list,
        (None, Some(one)) => Value::Array(vec![one]),
        _ => Value::Array(Vec::new()),
    })
}

fn pbs_datastores(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/pbs/assets/{}/details", ctx.arg(0));
    let data = ctx.get(&path)?;
    let list = datastore_list(&data)?;
    if ctx.json {
        return ctx.print_data(&list);
    }
    let rows: Vec<Datastore> = decode_rows(&list)
        .and_then(typed)
        .map_err(|e| CliError::Failed(format!("decode response data: datastores: {e}")))?;
    ctx.out.print(&render(
        &rows,
        &[
            col("STORE", |r: &Datastore| text(r.store.as_ref())),
            col("STATUS", |r: &Datastore| status_word(r.status.as_ref())),
            col("SIZE", |r: &Datastore| byte_size(r.total_bytes.as_ref())),
            col("USED", |r: &Datastore| byte_size(r.used_bytes.as_ref())),
            col("AVAIL", |r: &Datastore| byte_size(r.avail_bytes.as_ref())),
            col("USE%", |r: &Datastore| text(r.usage_percent.as_ref())),
            col("GROUPS", |r: &Datastore| text(r.group_count.as_ref())),
            col("SNAPSHOTS", |r: &Datastore| text(r.snapshot_count.as_ref())),
            col("LAST BACKUP", |r: &Datastore| {
                time_text(r.last_backup_at.as_ref())
            }),
        ],
    ));
    Ok(())
}

fn pbs_snapshots(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/pbs/assets/{}/snapshots", ctx.arg(0));
    let data = ctx.get(&path)?;
    ctx.print_list(
        &data,
        "snapshots",
        &[
            col("TYPE", |r: &Snapshot| text(r.backup_type.as_ref())),
            col("ID", |r: &Snapshot| text(r.backup_id.as_ref())),
            col("TIME", |r: &Snapshot| {
                unix_time_text(r.backup_time.as_ref())
            }),
            col("SIZE", |r: &Snapshot| byte_size(r.size.as_ref())),
        ],
    )
}

fn portainer_stacks(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/portainer/assets/{}/stacks", ctx.arg(0));
    let data = ctx.get(&path)?;
    ctx.print_list(
        &data,
        "data",
        &[
            col("ID", |r: &Stack| text(r.id.as_ref())),
            col("NAME", |r: &Stack| text(r.name.as_ref())),
            col("STATUS", |r: &Stack| stack_status(r.status.as_ref())),
        ],
    )
}

fn portainer_containers(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/portainer/assets/{}/containers", ctx.arg(0));
    let data = ctx.get(&path)?;
    ctx.print_list(
        &data,
        "data",
        &[
            col("ID", |r: &PortainerContainer| short_id(r.id.as_ref())),
            col("NAME", |r: &PortainerContainer| {
                container_names(r.names.as_ref())
            }),
            col("IMAGE", |r: &PortainerContainer| text(r.image.as_ref())),
            col("STATE", |r: &PortainerContainer| {
                status_word(r.state.as_ref())
            }),
            col("STATUS", |r: &PortainerContainer| text(r.status.as_ref())),
        ],
    )
}
