//! Rows printed by the device, agent, hub, search and credential commands.

use crate::output::cells::{EMPTY, status_word, text};
use crate::output::time::since_text;
use serde::Deserialize;
use serde_json::Value;

use super::at;

/// `GET /api/v2/assets` (a list of devices).
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Device {
    pub id: Option<Value>,
    pub platform: Option<Value>,
    pub name: Option<Value>,
    pub status: Option<Value>,
    pub last_seen_at: Option<Value>,
    pub metadata: Option<Value>,
}

impl Device {
    pub fn ip_address(&self) -> Option<&Value> {
        at(&self.metadata, "ip_address")
    }

    pub fn ip(&self) -> Option<&Value> {
        at(&self.metadata, "ip")
    }

    pub fn primary_ip(&self) -> Option<&Value> {
        at(&self.metadata, "primary_ip")
    }

    pub fn endpoint_host(&self) -> Option<&Value> {
        at(&self.metadata, "collector_endpoint_host")
    }
}

/// `GET /api/v2/agents`, list `assetsInfo`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Agent {
    pub id: Option<Value>,
    pub platform: Option<Value>,
}

/// `GET /api/v2/agents/pending`, list `agents`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct PendingAgent {
    pub asset_id: Option<Value>,
    pub hostname: Option<Value>,
    pub platform: Option<Value>,
    pub remote_ip: Option<Value>,
    pub connected_at: Option<Value>,
    pub identity_verified: Option<Value>,
}

/// `GET /api/v2/hub/agents`, list `presence`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Presence {
    pub asset_id: Option<Value>,
    pub transport: Option<Value>,
    pub connected_at: Option<Value>,
    pub last_heartbeat_at: Option<Value>,
}

/// `GET /api/v2/search`, lists `assets` and `groups`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct SearchResult {
    #[serde(rename = "type")]
    pub kind: Option<Value>,
    pub id: Option<Value>,
    pub name: Option<Value>,
    pub platform: Option<Value>,
    pub status: Option<Value>,
    pub last_seen_at: Option<Value>,
}

/// `GET /api/v2/credentials/profiles`, list `profiles`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct CredentialProfile {
    pub id: Option<Value>,
    pub name: Option<Value>,
    pub kind: Option<Value>,
    pub username: Option<Value>,
    pub status: Option<Value>,
}

/// The Go CLI's `deviceStatus`: "Offline since 21:12".
pub fn device_status(status: Option<&Value>, last_seen_at: Option<&Value>) -> String {
    let word = status_word(status);
    if word == "Offline" || word == "Not responding" {
        let since = last_seen_at
            .and_then(Value::as_str)
            .map(since_text)
            .unwrap_or_default();
        if !since.is_empty() {
            return format!("{word} since {since}");
        }
    }
    word
}

/// The Go CLI's `deviceAddress`: the first address the console would show.
pub fn device_address(device: &Device) -> String {
    [
        device.ip_address(),
        device.ip(),
        device.primary_ip(),
        device.endpoint_host(),
    ]
    .into_iter()
    .flatten()
    .filter_map(Value::as_str)
    .find(|text| !text.trim().is_empty())
    .map_or_else(|| EMPTY.to_string(), str::to_string)
}

/// The Go CLI's `resultKind`: the Hub calls a device an asset.
pub fn result_kind(value: Option<&Value>) -> String {
    let kind = text(value);
    if kind == "asset" {
        "device".to_string()
    } else {
        kind
    }
}
