//! `exec` (one device, or several with --targets or --group) and
//! `files cat`, with the Go CLI's checks, output and exit codes.

use std::collections::{BTreeMap, HashSet};

use labtether_cli_client::Method;
use serde_json::{Map, Value};

use super::Ctx;
use super::common::{as_object, str_member};
use crate::errors::CliError;

/// One command result from the Hub. Current Hubs send `status` and an
/// `exit_code` when the agent reported one; older ones send `exit_code`
/// only. A command that never ran has `error` and `message`.
#[derive(Debug, Default)]
struct ExecResult {
    status: String,
    exit_code: Option<i64>,
    stdout: String,
    error: String,
    message: String,
}

impl ExecResult {
    /// Go's decode into `remoteExecResult`; `exit_code` must be an integer.
    fn decode(value: Option<&Value>) -> Result<ExecResult, String> {
        let map: Option<&Map<String, Value>> = as_object(value)?;
        let exit_code = match map.and_then(|m| m.get("exit_code")) {
            None | Some(Value::Null) => None,
            Some(Value::Number(n)) => Some(
                n.as_i64()
                    .ok_or_else(|| format!("exit_code {n} is not a whole number"))?,
            ),
            Some(other) => {
                return Err(format!(
                    "exit_code is {}, not a number",
                    super::common::kind(other)
                ));
            }
        };
        Ok(ExecResult {
            status: str_member(map, "status")?,
            exit_code,
            stdout: str_member(map, "stdout")?,
            error: str_member(map, "error")?,
            message: str_member(map, "message")?,
        })
    }

    /// Whether the result says how the command ended at all.
    fn known(&self) -> bool {
        !self.error.is_empty() || !self.status.trim().is_empty() || self.exit_code.is_some()
    }

    fn failed(&self) -> bool {
        if !self.error.is_empty() {
            return true;
        }
        let status = self.status.trim();
        if !status.is_empty() {
            return !super::common::equal_fold(status, "succeeded");
        }
        self.exit_code != Some(0)
    }

    /// The Hub's reason for a command that never ran.
    fn reason(&self) -> String {
        let message = self.message.trim();
        if message.is_empty() {
            self.error.trim().to_string()
        } else {
            message.to_string()
        }
    }

    fn failure(&self) -> CliError {
        let (message, code) = if !self.error.is_empty() {
            (format!("remote command failed: {}", self.reason()), 1)
        } else if let Some(code) = self.exit_code {
            (
                format!("remote command failed with exit code {code}"),
                process_exit_code(code),
            )
        } else {
            (
                "remote command failed; the Hub did not report an exit code".to_string(),
                1,
            )
        };
        CliError::Remote { message, code }
    }
}

/// A remote exit code passes through when a process exit status can carry
/// it (1 to 255); anything else, such as a large Windows code, a negative
/// signal code or 0 on a failed result, becomes 1.
fn process_exit_code(code: i64) -> i32 {
    if (1..=255).contains(&code) {
        code as i32
    } else {
        1
    }
}

pub fn exec(ctx: &mut Ctx) -> Result<(), CliError> {
    let targets = ctx.flags.string("targets");
    let group = ctx.flags.string("group");
    let timeout = ctx.flags.int("timeout");
    if !(1..=300).contains(&timeout) {
        return Err(CliError::UsageInRun(
            "timeout must be between 1 and 300 seconds".into(),
        ));
    }
    let several = !targets.is_empty() || !group.is_empty();
    if !several && ctx.args.len() < 2 {
        return Err(CliError::UsageInRun(
            "a device and a command are required, or --targets or --group with a command".into(),
        ));
    }
    if several && ctx.args.is_empty() {
        return Err(CliError::UsageInRun("command is required".into()));
    }
    let client = ctx.client()?;
    if several {
        let mut pairs = vec![
            ("command", Value::String(ctx.args.join(" "))),
            ("timeout", Value::from(timeout)),
        ];
        if !targets.is_empty() {
            let list = targets
                .split(',')
                .map(|t| Value::String(t.into()))
                .collect();
            pairs.push(("targets", Value::Array(list)));
        }
        if !group.is_empty() {
            pairs.push(("group", Value::String(group)));
        }
        let data = client.send(Method::Post, "/api/v2/exec", Some(&Ctx::body(&pairs)))?;
        return several_results(ctx, &data, &targets);
    }
    let path = format!("/api/v2/assets/{}/exec", ctx.arg(0));
    let body = Ctx::body(&[
        ("command", Value::String(ctx.args[1..].join(" "))),
        ("timeout", Value::from(timeout)),
    ]);
    let data = client.send(Method::Post, &path, Some(&body))?;
    let result = ExecResult::decode(Some(&data))
        .map_err(|e| CliError::Failed(format!("failed to parse response: {e}")))?;
    if !result.known() {
        return Err(CliError::Failed(
            "remote command response is missing status and exit_code".into(),
        ));
    }
    if ctx.json {
        ctx.print_data(&data)?;
    } else if !result.stdout.is_empty() {
        ctx.out.print(&format!("{}\n", result.stdout));
    }
    if result.failed() {
        return Err(result.failure());
    }
    Ok(())
}

/// The `--targets` entries, trimmed and without repeats as the Hub reads
/// them, that have no result.
fn missing_targets(targets: &str, results: &BTreeMap<String, ExecResult>) -> Vec<String> {
    let mut seen = HashSet::new();
    targets
        .split(',')
        .map(str::trim)
        .filter(|t| !t.is_empty() && seen.insert(*t))
        .filter(|t| !results.contains_key(*t))
        .map(str::to_string)
        .collect()
}

fn several_results(ctx: &mut Ctx, data: &Value, targets: &str) -> Result<(), CliError> {
    let parse = |e: String| CliError::Failed(format!("failed to parse multi-target response: {e}"));
    let top = as_object(Some(data)).map_err(parse)?;
    let mut results = BTreeMap::new();
    match top.and_then(|t| t.get("results")) {
        None | Some(Value::Null) => {}
        Some(Value::Object(map)) => {
            for (target, value) in map {
                let result = ExecResult::decode(Some(value))
                    .map_err(|e| parse(format!("results {target:?}: {e}")))?;
                results.insert(target.clone(), result);
            }
        }
        Some(other) => {
            return Err(parse(format!(
                "results is {}, not an object",
                super::common::kind(other)
            )));
        }
    }
    if results.is_empty() {
        return Err(CliError::Failed(
            "multi-target response did not contain any results".into(),
        ));
    }
    if ctx.json {
        ctx.print_data(data)?;
    }
    // Sorted by target, as the Go CLI sorts byte-wise.
    if let Some((target, _)) = results.iter().find(|(_, r)| !r.known()) {
        return Err(CliError::Failed(format!(
            "multi-target result for {target} is missing status and exit_code"
        )));
    }
    // The Hub leaves out targets the API key cannot reach without saying
    // so; each requested target with no result is reported.
    let missing = missing_targets(targets, &results);
    let mut failed = missing.len();
    for (target, result) in &results {
        if result.failed() {
            failed += 1;
        }
        if ctx.json {
            continue;
        }
        let line = if !result.error.is_empty() {
            format!("[{target}] Error: {}\n", result.reason())
        } else if let (true, Some(code)) = (result.failed(), result.exit_code) {
            format!("[{target}] Exit code {code}: {}\n", result.stdout)
        } else if result.failed() {
            format!("[{target}] Failed: {}\n", result.stdout)
        } else {
            format!("[{target}] {}\n", result.stdout)
        };
        ctx.out.print(&line);
    }
    if !ctx.json {
        for target in &missing {
            ctx.out.print(&format!(
                "[{target}] Error: the hub returned no result for this device\n"
            ));
        }
    }
    if failed > 0 {
        return Err(CliError::Remote {
            message: format!(
                "{failed} of {} remote commands failed",
                results.len() + missing.len()
            ),
            code: 1,
        });
    }
    Ok(())
}

/// `files cat`: the file's bytes, written as they arrive.
pub fn files_cat(ctx: &mut Ctx) -> Result<(), CliError> {
    let client = ctx.client()?;
    let path = format!(
        "/api/v2/assets/{}/files/read?path={}",
        ctx.arg(0),
        ctx.query_arg(1)
    );
    Ok(client.download(&path, ctx.out.raw_writer())?)
}
