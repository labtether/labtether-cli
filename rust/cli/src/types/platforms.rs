//! Rows printed by the Proxmox, PBS and Portainer commands.

use crate::output::cells::{EMPTY, text};
use serde::Deserialize;
use serde_json::Value;

/// `GET /api/v2/proxmox/cluster/resources`, list `resources`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct ProxmoxResource {
    pub id: Option<Value>,
    #[serde(rename = "type")]
    pub kind: Option<Value>,
    pub node: Option<Value>,
    pub name: Option<Value>,
    pub status: Option<Value>,
    pub maxcpu: Option<Value>,
    pub maxmem: Option<Value>,
}

/// One datastore from `GET /api/v2/pbs/assets/{id}/details`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Datastore {
    pub store: Option<Value>,
    pub status: Option<Value>,
    pub total_bytes: Option<Value>,
    pub used_bytes: Option<Value>,
    pub avail_bytes: Option<Value>,
    pub usage_percent: Option<Value>,
    pub group_count: Option<Value>,
    pub snapshot_count: Option<Value>,
    pub last_backup_at: Option<Value>,
}

/// `GET /api/v2/pbs/assets/{id}/snapshots`, list `snapshots`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Snapshot {
    pub backup_type: Option<Value>,
    pub backup_id: Option<Value>,
    pub backup_time: Option<Value>,
    pub size: Option<Value>,
}

/// `GET /api/v2/portainer/assets/{id}/stacks`, list `data`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Stack {
    #[serde(rename = "Id")]
    pub id: Option<Value>,
    #[serde(rename = "Name")]
    pub name: Option<Value>,
    #[serde(rename = "Status")]
    pub status: Option<Value>,
}

/// `GET /api/v2/portainer/assets/{id}/containers`, list `data`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct PortainerContainer {
    #[serde(rename = "Id")]
    pub id: Option<Value>,
    #[serde(rename = "Names")]
    pub names: Option<Value>,
    #[serde(rename = "Image")]
    pub image: Option<Value>,
    #[serde(rename = "State")]
    pub state: Option<Value>,
    #[serde(rename = "Status")]
    pub status: Option<Value>,
}

/// The Go CLI's `portainerStackStatus`: 1 is active, 2 is inactive.
pub fn stack_status(value: Option<&Value>) -> String {
    match text(value).as_str() {
        "1" => "Active".to_string(),
        "2" => "Inactive".to_string(),
        other => other.to_string(),
    }
}

/// The Go CLI's `containerNames`: Docker names without the leading slash.
pub fn container_names(value: Option<&Value>) -> String {
    let Some(Value::Array(names)) = value else {
        return text(value);
    };
    names
        .iter()
        .map(|name| {
            let name = text(Some(name));
            name.strip_prefix('/').map(str::to_string).unwrap_or(name)
        })
        .filter(|name| name != EMPTY)
        .collect::<Vec<_>>()
        .join(", ")
}
