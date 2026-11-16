//! Commands aimed at one device or Docker host: cron, disks, net, users,
//! packages, services, ps, files, metrics, docker, exec and power.

use serde_json::Value;

use super::common::check_any_map;
use super::{Ctx, actions, exec, leaf, remote};
use crate::cobra::{Args, Command, Flag};
use crate::errors::CliError;
use crate::output::cells::{byte_size, short_id, text, time_text};
use crate::output::table::col;
use crate::types::device::*;

pub fn commands() -> Vec<Command> {
    vec![
        leaf("cron <device>", "Show cron jobs on a device", cron).exact(1),
        leaf("disks <device>", "Show disks and mounts on a device", disks).exact(1),
        leaf("net <device>", "Show network interfaces on a device", net).exact(1),
        leaf("users <device>", "Show who is logged in to a device", users).exact(1),
        Command::new("packages", "Install and update packages on a device").children(vec![
            leaf("list <device>", "List installed packages on a device", packages_list).exact(1),
            leaf("install <device> <package>", "Install a package on a device", actions::packages_install).exact(2),
            leaf("update <device>", "Update all packages on a device", actions::packages_update).exact(1),
        ]),
        Command::new("services", "List and control system services on a device").children(vec![
            leaf("list <device>", "List system services on a device", services_list).exact(1),
            leaf("start <device> <service>", "Start a service", actions::services_start).exact(2),
            leaf("stop <device> <service>", "Stop a service", actions::services_stop).exact(2),
            leaf("restart <device> <service>", "Restart a service", actions::services_restart).exact(2),
        ]),
        Command::new("ps", "List and signal processes on a device").children(vec![
            leaf("list <device>", "List running processes on a device", ps_list).exact(1),
            leaf("kill <device> <pid>", "Send a signal to a process on a device", actions::ps_kill)
                .exact(2)
                .flag(Flag::string("signal", "SIGTERM", "Signal to send, such as SIGTERM or SIGKILL")),
        ]),
        Command::new("files", "List and read files on a device").children(vec![
            leaf("ls <device> <path>", "List a directory on a device", files_ls).exact(2),
            leaf("cat <device> <path>", "Print a file from a device", exec::files_cat).exact(2),
        ]),
        Command::new("metrics", "Show device metrics").children(vec![
            leaf("overview", "Show metrics for all devices", |ctx| ctx.get_and_print("/api/v2/metrics/overview")),
            leaf("device <device>", "Show metrics for one device", |ctx| {
                let path = format!("/api/v2/assets/{}/metrics", ctx.arg(0));
                ctx.get_and_print(&path)
            })
            .aliases(&["asset"])
            .exact(1),
        ]),
        Command::new("docker", "List and control Docker containers").children(vec![
            leaf("hosts", "List Docker hosts", docker_hosts),
            leaf("ps <host>", "List containers on a Docker host", docker_ps)
                .exact(1)
                .flag(Flag::bool("all", "No effect; the list always includes stopped containers")),
            leaf("start <container>", "Start a container", remote::docker_start).exact(1),
            leaf("stop <container>", "Stop a container", remote::docker_stop).exact(1),
            leaf("restart <container>", "Restart a container", remote::docker_restart).exact(1),
            leaf("logs <container>", "Print a container's logs", docker_logs).exact(1).flag(Flag::bounded_int(
                "tail",
                100,
                "",
                1,
                10000,
                "Number of lines to show from the end of the logs",
            )),
        ]),
        leaf("exec <device> <command>", "Run a command on a device", exec::exec)
            .long(
                "Run a shell command on a device, or with --targets or --group on several devices. \
                 When the command has its own flags, quote it or put it after --.",
            )
            .example(
                "  labtether-cli exec nas01 uptime\n  labtether-cli exec nas01 -- df -h\n  labtether-cli exec --targets nas01,pve01 \"free -m\"",
            )
            .args(Args::Arbitrary)
            .flag(Flag::string("targets", "", "Device IDs to run on, separated by commas"))
            .flag(Flag::string("group", "", "Group ID to run on, as shown by groups list"))
            .flag(Flag::bounded_int("timeout", 30, "seconds", 1, 300, "Command timeout in seconds (max 300)")),
        leaf("reboot <device>", "Reboot a device", actions::reboot).exact(1),
        leaf("shutdown <device>", "Shut down a device", actions::shutdown).exact(1),
        leaf("wake <device>", "Wake a device with Wake-on-LAN", actions::wake).exact(1),
    ]
}

fn device_path(ctx: &Ctx, suffix: &str) -> String {
    format!("/api/v2/assets/{}/{suffix}", ctx.arg(0))
}

fn cron(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "cron"))?;
    ctx.print_list(
        &data,
        "entries",
        &[
            col("SOURCE", |r: &CronEntry| text(r.source.as_ref())),
            col("SCHEDULE", |r: &CronEntry| text(r.schedule.as_ref())),
            col("USER", |r: &CronEntry| text(r.user.as_ref())),
            col("COMMAND", |r: &CronEntry| text(r.command.as_ref())),
        ],
    )
}

fn disks(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "disks"))?;
    ctx.print_list(
        &data,
        "mounts",
        &[
            col("DEVICE", |r: &Mount| text(r.device.as_ref())),
            col("MOUNT", |r: &Mount| text(r.mount_point.as_ref())),
            col("TYPE", |r: &Mount| text(r.fs_type.as_ref())),
            col("SIZE", |r: &Mount| byte_size(r.total.as_ref())),
            col("USED", |r: &Mount| byte_size(r.used.as_ref())),
            col("AVAIL", |r: &Mount| byte_size(r.available.as_ref())),
            col("USE%", |r: &Mount| text(r.use_pct.as_ref())),
        ],
    )
}

fn net(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "network"))?;
    ctx.print_list(
        &data,
        "interfaces",
        &[
            col("NAME", |r: &Interface| text(r.name.as_ref())),
            col("STATE", |r: &Interface| text(r.state.as_ref())),
            col("MAC", |r: &Interface| text(r.mac.as_ref())),
            col("MTU", |r: &Interface| text(r.mtu.as_ref())),
            col("ADDRESSES", |r: &Interface| text(r.ips.as_ref())),
        ],
    )
}

fn users(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "users"))?;
    ctx.print_list(
        &data,
        "sessions",
        &[
            col("USER", |r: &Session| text(r.username.as_ref())),
            col("TERMINAL", |r: &Session| text(r.terminal.as_ref())),
            col("FROM", |r: &Session| text(r.remote_host.as_ref())),
            col("LOGIN", |r: &Session| time_text(r.login_time.as_ref())),
        ],
    )
}

fn packages_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "packages"))?;
    ctx.print_list(
        &data,
        "packages",
        &[
            col("NAME", |r: &Package| text(r.name.as_ref())),
            col("VERSION", |r: &Package| text(r.version.as_ref())),
            col("AVAILABLE", |r: &Package| {
                text(r.available_version.as_ref())
            }),
            col("STATUS", |r: &Package| text(r.status.as_ref())),
        ],
    )
}

fn services_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "services"))?;
    ctx.print_list(
        &data,
        "services",
        &[
            col("NAME", |r: &Service| text(r.name.as_ref())),
            col("ACTIVE", |r: &Service| text(r.active_state.as_ref())),
            col("SUB", |r: &Service| text(r.sub_state.as_ref())),
            col("ENABLED", |r: &Service| text(r.enabled.as_ref())),
            col("DESCRIPTION", |r: &Service| text(r.description.as_ref())),
        ],
    )
}

fn ps_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get(&device_path(ctx, "processes"))?;
    ctx.print_list(
        &data,
        "processes",
        &[
            col("PID", |r: &Process| text(r.pid.as_ref())),
            col("USER", |r: &Process| text(r.user.as_ref())),
            col("CPU%", |r: &Process| text(r.cpu_pct.as_ref())),
            col("MEM%", |r: &Process| text(r.mem_pct.as_ref())),
            col("COMMAND", |r: &Process| text(r.command.as_ref())),
        ],
    )
}

fn files_ls(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!(
        "/api/v2/assets/{}/files?path={}",
        ctx.arg(0),
        ctx.query_arg(1)
    );
    let data = ctx.get(&path)?;
    ctx.print_list(
        &data,
        "entries",
        &[
            col("TYPE", |r: &FileEntry| {
                if r.is_dir == Some(Value::Bool(true)) {
                    "directory"
                } else {
                    "file"
                }
                .to_string()
            }),
            col("SIZE", |r: &FileEntry| text(r.size.as_ref())),
            col("MODIFIED", |r: &FileEntry| time_text(r.mod_time.as_ref())),
            col("NAME", |r: &FileEntry| text(r.name.as_ref())),
        ],
    )
}

fn docker_hosts(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/docker/hosts")?;
    ctx.print_list(
        &data,
        "hosts",
        &[
            col("HOST", |r: &DockerHost| text(r.agent_id.as_ref())),
            col("OS", |r: &DockerHost| text(r.engine_os.as_ref())),
            col("CONTAINERS", |r: &DockerHost| {
                text(r.container_count.as_ref())
            }),
            col("VERSION", |r: &DockerHost| text(r.engine_version.as_ref())),
        ],
    )
}

fn docker_ps(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/docker/hosts/{}/containers", ctx.arg(0));
    let data = ctx.get(&path)?;
    ctx.print_list(
        &data,
        "containers",
        &[
            col("CONTAINER ID", |r: &Container| short_id(r.id.as_ref())),
            col("IMAGE", |r: &Container| text(r.image.as_ref())),
            col("STATUS", |r: &Container| text(r.status.as_ref())),
            col("PORTS", |r: &Container| text(r.ports.as_ref())),
            col("NAME", |r: &Container| text(r.name.as_ref())),
        ],
    )
}

/// `docker logs` prints the log text as it is, even with --json.
fn docker_logs(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!(
        "/api/v2/docker/containers/{}/logs?tail={}",
        ctx.arg(0),
        ctx.flags.int("tail")
    );
    let data = ctx.get(&path)?;
    let failed = |e: String| CliError::Failed(format!("decode response data: {e}"));
    check_any_map(&data).map_err(failed)?;
    if let Some(Value::String(logs)) = data.get("logs") {
        ctx.out.print(logs);
    }
    Ok(())
}
