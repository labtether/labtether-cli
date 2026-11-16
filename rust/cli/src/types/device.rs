//! Rows printed by the commands that read one device or Docker host.

use serde::Deserialize;
use serde_json::Value;

/// `GET /api/v2/assets/{id}/cron`, list `entries`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct CronEntry {
    pub source: Option<Value>,
    pub schedule: Option<Value>,
    pub user: Option<Value>,
    pub command: Option<Value>,
}

/// `GET /api/v2/assets/{id}/disks`, list `mounts`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Mount {
    pub device: Option<Value>,
    pub mount_point: Option<Value>,
    pub fs_type: Option<Value>,
    pub total: Option<Value>,
    pub used: Option<Value>,
    pub available: Option<Value>,
    pub use_pct: Option<Value>,
}

/// `GET /api/v2/assets/{id}/network`, list `interfaces`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Interface {
    pub name: Option<Value>,
    pub state: Option<Value>,
    pub mac: Option<Value>,
    pub mtu: Option<Value>,
    pub ips: Option<Value>,
}

/// `GET /api/v2/assets/{id}/users`, list `sessions`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Session {
    pub username: Option<Value>,
    pub terminal: Option<Value>,
    pub remote_host: Option<Value>,
    pub login_time: Option<Value>,
}

/// `GET /api/v2/assets/{id}/packages`, list `packages`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Package {
    pub name: Option<Value>,
    pub version: Option<Value>,
    pub available_version: Option<Value>,
    pub status: Option<Value>,
}

/// `GET /api/v2/assets/{id}/services`, list `services`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Service {
    pub name: Option<Value>,
    pub active_state: Option<Value>,
    pub sub_state: Option<Value>,
    pub enabled: Option<Value>,
    pub description: Option<Value>,
}

/// `GET /api/v2/assets/{id}/processes`, list `processes`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Process {
    pub pid: Option<Value>,
    pub user: Option<Value>,
    pub cpu_pct: Option<Value>,
    pub mem_pct: Option<Value>,
    pub command: Option<Value>,
}

/// `GET /api/v2/assets/{id}/files?path=...`, list `entries`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct FileEntry {
    pub is_dir: Option<Value>,
    pub size: Option<Value>,
    pub mod_time: Option<Value>,
    pub name: Option<Value>,
}

/// `GET /api/v2/docker/hosts`, list `hosts`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct DockerHost {
    pub agent_id: Option<Value>,
    pub engine_os: Option<Value>,
    pub container_count: Option<Value>,
    pub engine_version: Option<Value>,
}

/// `GET /api/v2/docker/hosts/{id}/containers`, list `containers`.
#[derive(Deserialize, Default)]
#[serde(default)]
pub struct Container {
    pub id: Option<Value>,
    pub image: Option<Value>,
    pub status: Option<Value>,
    pub ports: Option<Value>,
    pub name: Option<Value>,
}
