//! The shared command-to-route table (testdata/hub-routes.json), which the
//! Go CLI's tests read too. Every runnable command of the Rust tree must be
//! in it; every in-scope command must send the listed request; every
//! command not in the Rust build yet must refuse without a request.

mod common;

use common::{hub, run_against, temp_home};
use labtether_cli_rs::cobra::{CmdPath, Command, find::find};
use labtether_cli_rs::commands::{Run, root};
use serde_json::Value;

fn table() -> Value {
    let path = concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../testdata/hub-routes.json"
    );
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}

fn strings(value: &Value) -> Vec<String> {
    let Value::Array(items) = value else {
        panic!("not a list")
    };
    items
        .iter()
        .map(|v| v.as_str().unwrap().to_string())
        .collect()
}

fn list<'a>(table: &'a Value, key: &str) -> &'a [Value] {
    match table.get(key) {
        Some(Value::Array(items)) => items,
        _ => panic!("missing {key}"),
    }
}

fn runnable_paths(cmd: &Command, path: CmdPath, root: &Command, out: &mut Vec<(CmdPath, String)>) {
    for (i, child) in cmd.children.iter().enumerate() {
        let child_path = path.child(i);
        if child.runnable() && !child.hidden {
            let name = child_path
                .command_path(root)
                .trim_start_matches("labtether-cli ")
                .to_string();
            out.push((child_path.clone(), name));
        }
        runnable_paths(child, child_path, root, out);
    }
}

// Fault: a command was added to one CLI and not the other, or a Rust
// command sent a different method or path than the Go one.
#[test]
fn every_command_matches_the_shared_route_table() {
    let table = table();
    let root = root();
    let home = temp_home("routes");
    let mut covered = Vec::new();
    let (mut checked, mut refused) = (0, 0);
    for case in list(&table, "routes") {
        let name = case.get("name").and_then(Value::as_str).unwrap();
        let args = strings(case.get("args").unwrap());
        let (path, _, err) = find(&root, &args);
        assert!(
            err.is_none() && !path.is_root(),
            "{name}: no such command in the Rust tree"
        );
        covered.push(path.clone());
        let body = format!(
            "{{\"data\":{}}}",
            serde_json::to_string(case.get("data").unwrap()).unwrap()
        );
        let stand_in = hub(move |_| body.clone());
        let result = run_against(stand_in.port, &home, &args);
        let requests = stand_in.requests.lock().unwrap().clone();
        match path.resolve(&root).run {
            Some(Run::Rust(_)) => {
                let want = format!(
                    "{} {}",
                    case.get("method").and_then(Value::as_str).unwrap(),
                    case.get("path").and_then(Value::as_str).unwrap()
                );
                assert_eq!(
                    (result.code, requests.last()),
                    (0, Some(&want)),
                    "{name}: stderr {}",
                    result.stderr
                );
                checked += 1;
            }
            Some(Run::NotYet) => {
                assert_ne!(result.code, 0, "{name}");
                assert!(
                    result.stderr.contains("is not in the Rust build yet"),
                    "{name}: {}",
                    result.stderr
                );
                assert!(
                    requests.is_empty(),
                    "{name}: refused command sent {requests:?}"
                );
                refused += 1;
            }
            None => panic!("{name}: not runnable"),
        }
    }
    let local = strings(table.get("local").unwrap());
    let mut all = Vec::new();
    runnable_paths(&root, CmdPath::default(), &root, &mut all);
    for (path, name) in &all {
        assert!(
            covered.contains(path) || local.contains(name),
            "command {name:?} has no case in testdata/hub-routes.json"
        );
    }
    println!(
        "route cases: {checked} in the Rust build, {refused} not yet; runnable commands: {}",
        all.len()
    );
}

// Fault: a table command rejected a list shape the Hub sends (empty or
// null lists, extra fields).
#[test]
fn table_commands_accept_hub_list_envelopes() {
    let home = temp_home("envelopes");
    for case in list(&table(), "list_envelopes") {
        let name = case.get("name").and_then(Value::as_str).unwrap();
        let body = format!(
            "{{\"data\":{}}}",
            serde_json::to_string(case.get("data").unwrap()).unwrap()
        );
        let stand_in = hub(move |_| body.clone());
        let result = run_against(stand_in.port, &home, &strings(case.get("args").unwrap()));
        assert_eq!(result.code, 0, "{name}: {}", result.stderr);
    }
}

// Fault: an old 1.x command name stopped working or behaved differently.
#[test]
fn old_command_names_still_run() {
    let home = temp_home("renamed");
    let stand_in = hub(|request| {
        if request.ends_with(" /api/v2/assets") {
            r#"{"data":[{"id":"nas01","platform":"linux","status":"online","name":"nas01"}]}"#
                .to_string()
        } else if request.ends_with(" /api/v2/incidents") {
            r#"{"data":{"incidents":[{"id":"i1","severity":"high","status":"open","title":"Disk full"}]}}"#.to_string()
        } else {
            r#"{"data":{}}"#.to_string()
        }
    });
    for pair in list(&table(), "renamed") {
        let words = |key| {
            pair.get(key)
                .and_then(Value::as_str)
                .unwrap()
                .split(' ')
                .map(str::to_string)
                .collect::<Vec<_>>()
        };
        stand_in.requests.lock().unwrap().clear();
        let old = run_against(stand_in.port, &home, &words("old"));
        let current = run_against(stand_in.port, &home, &words("current"));
        let requests = stand_in.requests.lock().unwrap().clone();
        assert_eq!(
            (old.code, &old.stdout, &old.stderr),
            (current.code, &current.stdout, &current.stderr)
        );
        assert!(
            requests.len() < 2 || requests[0] == requests[1],
            "{requests:?}"
        );
    }
}
