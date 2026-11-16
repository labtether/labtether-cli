//! Commands that change something with one Hub request and print one line
//! (or the Hub's data with --json): power, agents, alerts, issues, groups,
//! Home Assistant, packages, services, processes, discovery, failover,
//! update plans and web services. Requests, bodies and messages are the Go
//! CLI's.

use labtether_cli_client::Method;
use serde_json::Value;

use super::Ctx;
use crate::errors::CliError;

/// The Go CLI's long-action client timeout: package actions can run for 10
/// minutes on the Hub.
pub const LONG_ACTION: std::time::Duration = std::time::Duration::from_secs(12 * 60);

fn text(value: &str) -> Value {
    Value::String(value.to_string())
}

/// POST with no body, then the Go CLI's `printMutationResult`.
fn post_said(ctx: &mut Ctx, path: &str, said: String) -> Result<(), CliError> {
    let data = ctx.send(Method::Post, path, None)?;
    ctx.print_status(&data, &said)
}

pub fn reboot(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/assets/{}/reboot", ctx.arg(0));
    let said = format!("Reboot requested for {}\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

pub fn shutdown(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/assets/{}/shutdown", ctx.arg(0));
    let said = format!("Shutdown requested for {}\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

pub fn wake(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/assets/{}/wake", ctx.arg(0));
    let said = format!("Wake-on-LAN sent to {}\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

fn pending_agent(ctx: &mut Ctx, verb: &str, done: &str) -> Result<(), CliError> {
    let body = Ctx::body(&[("asset_id", text(&ctx.args[0]))]);
    let data = ctx.send(
        Method::Post,
        &format!("/api/v2/agents/pending/{verb}"),
        Some(body),
    )?;
    let said = format!("Agent {} {done}\n", ctx.args[0]);
    ctx.print_status(&data, &said)
}

pub fn agents_approve(ctx: &mut Ctx) -> Result<(), CliError> {
    pending_agent(ctx, "approve", "approved")
}

pub fn agents_reject(ctx: &mut Ctx) -> Result<(), CliError> {
    pending_agent(ctx, "reject", "rejected")
}

pub fn alerts_ack(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/alerts/{}/ack", ctx.arg(0));
    let said = format!("Alert {} acknowledged\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

pub fn alerts_resolve(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/alerts/{}/resolve", ctx.arg(0));
    let said = format!("Alert {} resolved\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

fn issue_status(ctx: &mut Ctx, status: &str, said: String) -> Result<(), CliError> {
    let path = format!("/api/v2/incidents/{}", ctx.arg(0));
    let body = Ctx::body(&[("status", text(status))]);
    let data = ctx.send(Method::Patch, &path, Some(body))?;
    ctx.print_status(&data, &said)
}

pub fn issues_investigate(ctx: &mut Ctx) -> Result<(), CliError> {
    let said = format!("Issue {} moved to Investigating\n", ctx.args[0]);
    issue_status(ctx, "investigating", said)
}

pub fn issues_resolve(ctx: &mut Ctx) -> Result<(), CliError> {
    let said = format!("Issue {} resolved\n", ctx.args[0]);
    issue_status(ctx, "resolved", said)
}

/// `groups create` prints the Hub's data as JSON, with or without --json.
pub fn groups_create(ctx: &mut Ctx) -> Result<(), CliError> {
    let name = ctx.flags.string("name");
    if name.is_empty() {
        return Err(CliError::UsageInRun("--name is required".into()));
    }
    let body = Ctx::body(&[("name", text(&name))]);
    let data = ctx.send(Method::Post, "/api/v2/groups", Some(body))?;
    ctx.print_data(&data)
}

pub fn ha_call(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/homeassistant/entities/{}", ctx.arg(0));
    let body = Ctx::body(&[
        ("action", text("service.call")),
        ("service", text(&ctx.args[1])),
    ]);
    let data = ctx.send(Method::Post, &path, Some(body))?;
    let said = format!("Called service {} on entity {}\n", ctx.args[1], ctx.args[0]);
    ctx.print_status(&data, &said)
}

pub fn packages_install(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/assets/{}/packages/install", ctx.arg(0));
    let body = Ctx::body(&[("package", text(&ctx.args[1]))]);
    let client = ctx.client_with(LONG_ACTION)?;
    let data = client.send(Method::Post, &path, Some(&body))?;
    let said = format!(
        "Package {} install requested on {}\n",
        ctx.args[1], ctx.args[0]
    );
    ctx.print_status(&data, &said)
}

pub fn packages_update(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/assets/{}/packages/upgrade", ctx.arg(0));
    let body = Ctx::body(&[]);
    let client = ctx.client_with(LONG_ACTION)?;
    let data = client.send(Method::Post, &path, Some(&body))?;
    let said = format!("Package update requested on {}\n", ctx.args[0]);
    ctx.print_status(&data, &said)
}

fn service_action(ctx: &mut Ctx, verb: &str, done: &str) -> Result<(), CliError> {
    let path = format!(
        "/api/v2/assets/{}/services/{}/{verb}",
        ctx.arg(0),
        ctx.arg(1)
    );
    let said = format!("Service {} {done} on {}\n", ctx.args[1], ctx.args[0]);
    post_said(ctx, &path, said)
}

pub fn services_start(ctx: &mut Ctx) -> Result<(), CliError> {
    service_action(ctx, "start", "started")
}

pub fn services_stop(ctx: &mut Ctx) -> Result<(), CliError> {
    service_action(ctx, "stop", "stopped")
}

pub fn services_restart(ctx: &mut Ctx) -> Result<(), CliError> {
    service_action(ctx, "restart", "restarted")
}

pub fn ps_kill(ctx: &mut Ctx) -> Result<(), CliError> {
    // Go's strconv.Atoi: an optional sign, then ASCII digits, as i64 parses.
    let pid = match ctx.args[1].parse::<i64>() {
        Ok(pid) if pid > 0 => pid,
        _ => {
            return Err(CliError::UsageInRun(
                "pid must be a positive integer".into(),
            ));
        }
    };
    let client = ctx.client()?;
    let signal = ctx.flags.string("signal");
    let body = Ctx::body(&[("pid", Value::from(pid)), ("signal", text(&signal))]);
    let path = format!("/api/v2/assets/{}/processes/kill", ctx.arg(0));
    let data = client.send(Method::Post, &path, Some(&body))?;
    let said = format!(
        "Signal {signal} sent to PID {} on {}\n",
        ctx.args[1], ctx.args[0]
    );
    ctx.print_status(&data, &said)
}

pub fn discovery_run(ctx: &mut Ctx) -> Result<(), CliError> {
    post_said(
        ctx,
        "/api/v2/discovery/run",
        "Discovery scan started\n".into(),
    )
}

pub fn discovery_accept(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/discovery/proposals/{}/accept", ctx.arg(0));
    let said = format!("Suggested relationship {} accepted\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

pub fn discovery_dismiss(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/discovery/proposals/{}/dismiss", ctx.arg(0));
    let said = format!("Suggested relationship {} dismissed\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

/// `failover check-readiness` prints the Hub's data as JSON.
pub fn failover_readiness(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/failover-pairs/{}/check-readiness", ctx.arg(0));
    let data = ctx.send(Method::Post, &path, None)?;
    ctx.print_data(&data)
}

pub fn update_plan_execute(ctx: &mut Ctx) -> Result<(), CliError> {
    let path = format!("/api/v2/updates/plans/{}/execute", ctx.arg(0));
    let said = format!("Update plan {} started\n", ctx.args[0]);
    post_said(ctx, &path, said)
}

pub fn web_services_sync(ctx: &mut Ctx) -> Result<(), CliError> {
    post_said(
        ctx,
        "/api/v2/web-services/sync",
        "Web services sync started\n".into(),
    )
}
