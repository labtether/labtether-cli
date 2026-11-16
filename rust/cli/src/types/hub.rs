//! Rows printed by the connector, collector, group, web service and Home
//! Assistant commands.

use serde::Deserialize;
use serde_json::Value;

use super::at;

/// `GET /api/v2/connectors`, list `connectors`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Connector {
    pub id: Option<Value>,
    pub display_name: Option<Value>,
}

/// `GET /api/v2/collectors`, list `collectors`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Collector {
    pub id: Option<Value>,
    pub collector_type: Option<Value>,
    pub asset_id: Option<Value>,
    pub enabled: Option<Value>,
    pub last_status: Option<Value>,
    pub last_collected_at: Option<Value>,
}

/// `GET /api/v2/groups`, list `groups`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Group {
    pub id: Option<Value>,
    pub name: Option<Value>,
    pub slug: Option<Value>,
    pub parent_group_id: Option<Value>,
}

/// `GET /api/v2/web-services`, list `services`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct WebService {
    pub name: Option<Value>,
    pub category: Option<Value>,
    pub status: Option<Value>,
    pub host_asset_id: Option<Value>,
    pub url: Option<Value>,
}

/// `GET /api/v2/homeassistant/entities`, list `entities`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Entity {
    pub id: Option<Value>,
    pub metadata: Option<Value>,
    pub name: Option<Value>,
    pub status: Option<Value>,
}

impl Entity {
    pub fn entity_id(&self) -> Option<&Value> {
        at(&self.metadata, "entity_id")
    }

    pub fn state(&self) -> Option<&Value> {
        at(&self.metadata, "state")
    }
}
