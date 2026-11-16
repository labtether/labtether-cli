//! Rows printed by the alert, issue, audit, check, notification, failover,
//! discovery and update commands.

use serde::Deserialize;
use serde_json::Value;

use super::at;

/// `GET /api/v2/alerts`, list `instances`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct AlertInstance {
    pub id: Option<Value>,
    pub severity: Option<Value>,
    pub status: Option<Value>,
    pub started_at: Option<Value>,
    pub rule_id: Option<Value>,
    pub firing_scope: Option<Value>,
}

impl AlertInstance {
    pub fn asset_ids(&self) -> Option<&Value> {
        at(&self.firing_scope, "asset_ids")
    }
}

/// `GET /api/v2/incidents`, list `incidents`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Incident {
    pub id: Option<Value>,
    pub severity: Option<Value>,
    pub status: Option<Value>,
    pub opened_at: Option<Value>,
    pub title: Option<Value>,
}

/// `GET /api/v2/audit/events`, list `events`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct AuditEvent {
    pub timestamp: Option<Value>,
    #[serde(rename = "type")]
    pub kind: Option<Value>,
    pub actor_id: Option<Value>,
    pub target: Option<Value>,
    pub decision: Option<Value>,
}

/// `GET /api/v2/synthetic-checks`, list `checks`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Check {
    pub id: Option<Value>,
    pub name: Option<Value>,
    pub check_type: Option<Value>,
    pub enabled: Option<Value>,
    pub last_status: Option<Value>,
    pub target: Option<Value>,
}

/// `GET /api/v2/notifications/channels`, list `channels`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Channel {
    pub id: Option<Value>,
    pub name: Option<Value>,
    #[serde(rename = "type")]
    pub kind: Option<Value>,
    pub enabled: Option<Value>,
}

/// `GET /api/v2/notifications/history`, list `history`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Notification {
    pub id: Option<Value>,
    pub channel_id: Option<Value>,
    pub status: Option<Value>,
    pub sent_at: Option<Value>,
    pub retry_count: Option<Value>,
    pub error: Option<Value>,
}

/// `GET /api/v2/failover-pairs`, list `pairs`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct FailoverPair {
    pub id: Option<Value>,
    pub name: Option<Value>,
    pub primary_group_id: Option<Value>,
    pub backup_group_id: Option<Value>,
    pub readiness_score: Option<Value>,
}

/// `GET /api/v2/discovery/proposals`, list `proposals`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Proposal {
    pub id: Option<Value>,
    pub source_asset_id: Option<Value>,
    pub target_asset_id: Option<Value>,
    pub relationship_type: Option<Value>,
    pub confidence: Option<Value>,
}

/// `GET /api/v2/updates/plans`, list `plans`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct UpdatePlan {
    pub id: Option<Value>,
    pub name: Option<Value>,
    pub default_dry_run: Option<Value>,
    pub targets: Option<Value>,
}

/// `GET /api/v2/updates/runs`, list `runs`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct UpdateRun {
    pub id: Option<Value>,
    pub plan_name: Option<Value>,
    pub status: Option<Value>,
    pub dry_run: Option<Value>,
    pub created_at: Option<Value>,
}
