//! connectors, collectors, groups, web-services, ha and map.

use serde::Deserialize;
use serde_json::Value;

use super::monitoring::by_id;
use super::{Ctx, leaf, not_yet};
use crate::cobra::quote::quote;
use crate::cobra::{Command, Flag};
use crate::errors::CliError;
use crate::output::cells::{status_word, text, time_text};
use crate::output::object;
use crate::output::table::col;
use crate::types::hub::*;

pub fn commands() -> Vec<Command> {
    vec![
        Command::new("connectors", "List and test hub connectors").children(vec![
            leaf("list", "List connectors", connectors_list),
            leaf("get <id>", "Show one connector", connectors_get).exact(1),
            not_yet("test <connector-or-collector-id>", "Test a saved connection")
                .long(
                    "Test the saved connection for a connector, such as proxmox or truenas. \
                     When a connector has more than one saved connection, give its collector ID, as shown by collectors list.",
                )
                .exact(1),
        ]),
        Command::new("collectors", "Show the hub's collectors").children(vec![
            leaf("list", "List collectors", collectors_list),
            leaf("get <id>", "Show one collector", |ctx| by_id(ctx, "/api/v2/collectors/")).exact(1),
        ]),
        Command::new("groups", "List and create device groups").children(vec![
            leaf("list", "List groups", groups_list),
            leaf("get <group>", "Show one group", |ctx| by_id(ctx, "/api/v2/groups/")).exact(1),
            not_yet("create", "Create a group").flag(Flag::string("name", "", "Group name (required)")),
        ]),
        Command::new("web-services", "List and sync the web services the hub found").children(vec![
            leaf("list", "List web services", web_services_list),
            not_yet("sync", "Start a web services sync"),
        ]),
        Command::new("ha", "List and control Home Assistant entities").children(vec![
            leaf("entities", "List Home Assistant entities", ha_entities),
            leaf("entity <id>", "Show one Home Assistant entity, by the ID from ha entities", |ctx| {
                by_id(ctx, "/api/v2/homeassistant/entities/")
            })
            .exact(1),
            not_yet("call <id> <service>", "Call a Home Assistant service, such as switch.turn_on, on an entity").exact(2),
        ]),
        Command::new("map", "Show how devices connect")
            .aliases(&["topology"])
            .example("  labtether-cli map connections pve01\n  labtether-cli map downstream pve01\n  labtether-cli map upstream nas01")
            .children(vec![
                leaf("connections <device>", "List a device's connections", |ctx| map_query(ctx, "/api/v2/edges", "asset_id"))
                    .aliases(&["edges"])
                    .exact(1),
                leaf("dependencies <device>", "List a device's service dependencies", |ctx| {
                    map_query(ctx, "/api/v2/dependencies", "asset_id")
                })
                .exact(1),
                leaf("downstream <device>", "Show the connections downstream of a device", |ctx| {
                    map_query(ctx, "/api/v2/edges/tree", "root")
                })
                .aliases(&["blast-radius"])
                .exact(1),
                leaf("upstream <device>", "Show the connections upstream of a device", |ctx| {
                    map_query(ctx, "/api/v2/edges/ancestors", "id")
                })
                .exact(1),
            ]),
    ]
}

fn map_query(ctx: &mut Ctx, route: &str, param: &str) -> Result<(), CliError> {
    let path = format!("{route}?{param}={}", ctx.query_arg(0));
    ctx.get_and_print(&path)
}

fn connectors_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/connectors")?;
    ctx.print_list(
        &data,
        "connectors",
        &[
            col("ID", |r: &Connector| text(r.id.as_ref())),
            col("NAME", |r: &Connector| text(r.display_name.as_ref())),
        ],
    )
}

/// One connector descriptor as the Go CLI's struct holds it.
#[derive(Deserialize, Default)]
#[serde(default)]
struct Descriptor {
    id: Option<String>,
    display_name: Option<String>,
    capabilities: Option<Value>,
}

#[derive(Deserialize)]
struct ConnectorList {
    #[serde(default)]
    connectors: Option<Vec<Option<Descriptor>>>,
}

/// The Go CLI's decode of `{"connectors": [{id, display_name,
/// capabilities}]}`: `None` where it fails or the list is null.
fn descriptors(data: &Value) -> Option<Vec<Descriptor>> {
    if !data.is_object() {
        return None;
    }
    let list = ConnectorList::deserialize(data).ok()?.connectors?;
    Some(list.into_iter().map(Option::unwrap_or_default).collect())
}

fn connectors_get(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/connectors")?;
    let list = descriptors(&data)
        .ok_or_else(|| CliError::Failed("decode connector list: invalid response".into()))?;
    let wanted = &ctx.args[0];
    match list
        .into_iter()
        .find(|d| d.id.as_deref().unwrap_or_default() == wanted)
    {
        Some(found) => {
            let value = object(&[
                ("id", Value::String(found.id.unwrap_or_default())),
                (
                    "display_name",
                    Value::String(found.display_name.unwrap_or_default()),
                ),
                ("capabilities", found.capabilities.unwrap_or(Value::Null)),
            ]);
            ctx.print_data(&value)
        }
        None => Err(CliError::NotFound(format!(
            "connector {} not found",
            quote(wanted)
        ))),
    }
}

fn collectors_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/collectors")?;
    ctx.print_list(
        &data,
        "collectors",
        &[
            col("ID", |r: &Collector| text(r.id.as_ref())),
            col("TYPE", |r: &Collector| text(r.collector_type.as_ref())),
            col("DEVICE", |r: &Collector| text(r.asset_id.as_ref())),
            col("ENABLED", |r: &Collector| text(r.enabled.as_ref())),
            col("LAST STATUS", |r: &Collector| {
                status_word(r.last_status.as_ref())
            }),
            col("LAST RUN", |r: &Collector| {
                time_text(r.last_collected_at.as_ref())
            }),
        ],
    )
}

fn groups_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/groups")?;
    ctx.print_list(
        &data,
        "groups",
        &[
            col("ID", |r: &Group| text(r.id.as_ref())),
            col("NAME", |r: &Group| text(r.name.as_ref())),
            col("SLUG", |r: &Group| text(r.slug.as_ref())),
            col("PARENT", |r: &Group| text(r.parent_group_id.as_ref())),
        ],
    )
}

fn web_services_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/web-services")?;
    ctx.print_list(
        &data,
        "services",
        &[
            col("NAME", |r: &WebService| text(r.name.as_ref())),
            col("CATEGORY", |r: &WebService| text(r.category.as_ref())),
            col("STATUS", |r: &WebService| status_word(r.status.as_ref())),
            col("HOST", |r: &WebService| text(r.host_asset_id.as_ref())),
            col("URL", |r: &WebService| text(r.url.as_ref())),
        ],
    )
}

fn ha_entities(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/homeassistant/entities")?;
    ctx.print_list(
        &data,
        "entities",
        &[
            col("ID", |r: &Entity| text(r.id.as_ref())),
            col("ENTITY", |r: &Entity| text(r.entity_id())),
            col("NAME", |r: &Entity| text(r.name.as_ref())),
            col("STATE", |r: &Entity| text(r.state())),
            col("STATUS", |r: &Entity| status_word(r.status.as_ref())),
        ],
    )
}
