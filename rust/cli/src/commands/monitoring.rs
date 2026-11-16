//! alerts, issues, audit, checks, notifications, failover, discovery and
//! updates.

use std::collections::HashMap;

use serde_json::Value;

use super::common::{decode_list, rows};
use super::{Ctx, leaf, not_yet};
use crate::cobra::Command;
use crate::errors::CliError;
use crate::output::cells::{status_word, text, time_text};
use crate::output::table::{col, render};
use crate::types::monitoring::*;

fn get_leaf(usage: &'static str, short: &str, handler: super::Handler) -> Command {
    leaf(usage, short, handler).exact(1)
}

pub fn commands() -> Vec<Command> {
    vec![
        Command::new("alerts", "List and resolve alerts").children(vec![
            leaf("list", "List active alerts", alerts_list),
            not_yet("ack <alert>", "Acknowledge an alert").exact(1),
            not_yet("resolve <alert>", "Resolve an alert").exact(1),
        ]),
        Command::new("issues", "List and resolve issues")
            .aliases(&["incidents"])
            .example("  labtether-cli issues list\n  labtether-cli issues investigate <id>\n  labtether-cli issues resolve <id>")
            .children(vec![
                leaf("list", "List issues", issues_list),
                get_leaf("get <id>", "Show one issue", |ctx| by_id(ctx, "/api/v2/incidents/")),
                not_yet("investigate <id>", "Move an issue to Investigating").aliases(&["ack"]).exact(1),
                not_yet("resolve <id>", "Resolve an issue").exact(1),
            ]),
        leaf("audit", "Show the audit log", audit),
        Command::new("checks", "Show service checks").children(vec![
            leaf("list", "List service checks", checks_list),
            get_leaf("get <id>", "Show one check", |ctx| by_id(ctx, "/api/v2/synthetic-checks/")),
        ]),
        Command::new("notifications", "Show notification channels and history").children(vec![
            leaf("channels", "List notification channels", notification_channels),
            leaf("history", "List sent notifications", notification_history),
        ]),
        Command::new("failover", "Show failover pairs").children(vec![
            leaf("list", "List failover pairs", failover_list),
            get_leaf("get <id>", "Show one failover pair", |ctx| by_id(ctx, "/api/v2/failover-pairs/")),
            not_yet("check-readiness <id>", "Check whether a failover pair is ready").exact(1),
        ]),
        Command::new("discovery", "Run discovery and review suggested relationships").children(vec![
            not_yet("run", "Start a discovery scan"),
            leaf("proposals", "List suggested relationships", discovery_proposals),
            not_yet("accept <id>", "Accept a suggested relationship").exact(1),
            not_yet("dismiss <id>", "Dismiss a suggested relationship").aliases(&["reject"]).exact(1),
        ]),
        Command::new("updates", "Run update plans and list their runs").children(vec![
            Command::new("plans", "List and run update plans").children(vec![
                leaf("list", "List update plans", update_plans),
                not_yet("execute <plan>", "Run an update plan").exact(1),
            ]),
            Command::new("runs", "Show past update runs").children(vec![
                leaf("list", "List update runs", update_runs),
                get_leaf("get <run>", "Show one update run", |ctx| by_id(ctx, "/api/v2/updates/runs/")),
            ]),
        ]),
    ]
}

/// GET `<prefix><escaped id>` and print its payload.
pub fn by_id(ctx: &mut Ctx, prefix: &str) -> Result<(), CliError> {
    let path = format!("{prefix}{}", ctx.arg(0));
    ctx.get_and_print(&path)
}

fn alerts_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let client = ctx.client()?;
    let data = client.get("/api/v2/alerts")?;
    if ctx.json {
        return ctx.print_data(&data);
    }
    let rows: Vec<AlertInstance> = rows(&data, "instances")?;
    // Rule names make the list say what fired; without access to the rules
    // the list shows rule IDs.
    let mut names: HashMap<String, String> = HashMap::new();
    if let Ok(rules) = client.get("/api/v2/alerts/rules")
        && let Ok(rows) = decode_list(&rules, "rules")
    {
        for row in rows {
            if let (Some(Value::String(id)), Some(Value::String(name))) =
                (row.get("id"), row.get("name"))
            {
                names.insert(id.clone(), name.trim().to_string());
            }
        }
    }
    let rule = |row: &AlertInstance| {
        let id = text(row.rule_id.as_ref());
        names
            .get(&id)
            .filter(|name| !name.is_empty())
            .cloned()
            .unwrap_or(id)
    };
    let rule_cells: Vec<String> = rows.iter().map(rule).collect();
    let rows: Vec<(AlertInstance, String)> = rows.into_iter().zip(rule_cells).collect();
    ctx.out.print(&render(
        &rows,
        &[
            col("ID", |(r, _): &(AlertInstance, String)| text(r.id.as_ref())),
            col("SEVERITY", |(r, _): &(AlertInstance, String)| {
                status_word(r.severity.as_ref())
            }),
            col("STATE", |(r, _): &(AlertInstance, String)| {
                status_word(r.status.as_ref())
            }),
            col("STARTED", |(r, _): &(AlertInstance, String)| {
                time_text(r.started_at.as_ref())
            }),
            col("RULE", |(_, rule): &(AlertInstance, String)| rule.clone()),
            col("DEVICE", |(r, _): &(AlertInstance, String)| {
                text(r.asset_ids())
            }),
        ],
    ));
    Ok(())
}

fn issues_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/incidents")?;
    ctx.print_list(
        &data,
        "incidents",
        &[
            col("ID", |r: &Incident| text(r.id.as_ref())),
            col("SEVERITY", |r: &Incident| status_word(r.severity.as_ref())),
            col("STATE", |r: &Incident| status_word(r.status.as_ref())),
            col("OPENED", |r: &Incident| time_text(r.opened_at.as_ref())),
            col("TITLE", |r: &Incident| text(r.title.as_ref())),
        ],
    )
}

fn audit(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/audit/events")?;
    ctx.print_list(
        &data,
        "events",
        &[
            col("TIME", |r: &AuditEvent| time_text(r.timestamp.as_ref())),
            col("TYPE", |r: &AuditEvent| text(r.kind.as_ref())),
            col("ACTOR", |r: &AuditEvent| text(r.actor_id.as_ref())),
            col("TARGET", |r: &AuditEvent| text(r.target.as_ref())),
            col("DECISION", |r: &AuditEvent| text(r.decision.as_ref())),
        ],
    )
}

fn checks_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/synthetic-checks")?;
    ctx.print_list(
        &data,
        "checks",
        &[
            col("ID", |r: &Check| text(r.id.as_ref())),
            col("NAME", |r: &Check| text(r.name.as_ref())),
            col("TYPE", |r: &Check| text(r.check_type.as_ref())),
            col("ENABLED", |r: &Check| text(r.enabled.as_ref())),
            col("LAST STATUS", |r: &Check| {
                status_word(r.last_status.as_ref())
            }),
            col("TARGET", |r: &Check| text(r.target.as_ref())),
        ],
    )
}

fn notification_channels(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/notifications/channels")?;
    ctx.print_list(
        &data,
        "channels",
        &[
            col("ID", |r: &Channel| text(r.id.as_ref())),
            col("NAME", |r: &Channel| text(r.name.as_ref())),
            col("TYPE", |r: &Channel| text(r.kind.as_ref())),
            col("ENABLED", |r: &Channel| text(r.enabled.as_ref())),
        ],
    )
}

fn notification_history(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/notifications/history")?;
    ctx.print_list(
        &data,
        "history",
        &[
            col("ID", |r: &Notification| text(r.id.as_ref())),
            col("CHANNEL", |r: &Notification| text(r.channel_id.as_ref())),
            col("STATUS", |r: &Notification| status_word(r.status.as_ref())),
            col("SENT", |r: &Notification| time_text(r.sent_at.as_ref())),
            col("RETRIES", |r: &Notification| text(r.retry_count.as_ref())),
            col("ERROR", |r: &Notification| text(r.error.as_ref())),
        ],
    )
}

fn failover_list(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/failover-pairs")?;
    ctx.print_list(
        &data,
        "pairs",
        &[
            col("ID", |r: &FailoverPair| text(r.id.as_ref())),
            col("NAME", |r: &FailoverPair| text(r.name.as_ref())),
            col("PRIMARY", |r: &FailoverPair| {
                text(r.primary_group_id.as_ref())
            }),
            col("BACKUP", |r: &FailoverPair| {
                text(r.backup_group_id.as_ref())
            }),
            col("READINESS", |r: &FailoverPair| {
                text(r.readiness_score.as_ref())
            }),
        ],
    )
}

fn discovery_proposals(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/discovery/proposals")?;
    ctx.print_list(
        &data,
        "proposals",
        &[
            col("ID", |r: &Proposal| text(r.id.as_ref())),
            col("SOURCE", |r: &Proposal| text(r.source_asset_id.as_ref())),
            col("TARGET", |r: &Proposal| text(r.target_asset_id.as_ref())),
            col("RELATIONSHIP", |r: &Proposal| {
                text(r.relationship_type.as_ref())
            }),
            col("CONFIDENCE", |r: &Proposal| text(r.confidence.as_ref())),
        ],
    )
}

fn update_plans(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/updates/plans")?;
    ctx.print_list(
        &data,
        "plans",
        &[
            col("ID", |r: &UpdatePlan| text(r.id.as_ref())),
            col("NAME", |r: &UpdatePlan| text(r.name.as_ref())),
            col("DRY RUN", |r: &UpdatePlan| text(r.default_dry_run.as_ref())),
            col("TARGETS", |r: &UpdatePlan| text(r.targets.as_ref())),
        ],
    )
}

fn update_runs(ctx: &mut Ctx) -> Result<(), CliError> {
    let data = ctx.get("/api/v2/updates/runs")?;
    ctx.print_list(
        &data,
        "runs",
        &[
            col("ID", |r: &UpdateRun| text(r.id.as_ref())),
            col("PLAN", |r: &UpdateRun| text(r.plan_name.as_ref())),
            col("STATUS", |r: &UpdateRun| status_word(r.status.as_ref())),
            col("DRY RUN", |r: &UpdateRun| text(r.dry_run.as_ref())),
            col("CREATED", |r: &UpdateRun| time_text(r.created_at.as_ref())),
        ],
    )
}
