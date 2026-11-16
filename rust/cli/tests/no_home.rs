//! Runs without a home folder, through the built binary so the settings come
//! from the real environment: the CLI must not read or write settings in the
//! shared temporary folder, and a run with LABTETHER_HOST and
//! LABTETHER_API_KEY must still work.

mod common;

use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Output, Stdio};

/// A temporary folder for one test, standing in for the system one.
fn shared_temp(name: &str) -> PathBuf {
    let dir = std::env::temp_dir().join(format!("ltcli-nohome-{name}-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

/// Runs the binary with no home folder and `temp` as the temporary folder.
fn run(temp: &Path, env: &[(&str, &str)], args: &[&str], stdin: &str) -> Output {
    let mut command = Command::new(env!("CARGO_BIN_EXE_labtether-cli-rs"));
    for name in [
        "HOME",
        "USERPROFILE",
        "LABTETHER_HOST",
        "LABTETHER_API_KEY",
        "LABTETHER_TLS_CA_FILE",
    ] {
        command.env_remove(name);
    }
    for name in ["TMPDIR", "TMP", "TEMP"] {
        command.env(name, temp);
    }
    command.envs(env.iter().copied()).args(args);
    command
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    let mut child = command.spawn().unwrap();
    child
        .stdin
        .take()
        .unwrap()
        .write_all(stdin.as_bytes())
        .unwrap();
    child.wait_with_output().unwrap()
}

// Fault: with no home folder the CLI read its config from
// <temp folder>/labtether, so a config another local user put there sent
// the API key from LABTETHER_API_KEY to that user's Hub address.
#[test]
fn a_config_in_the_shared_temp_folder_is_not_read() {
    let temp = shared_temp("read");
    let planted = temp.join("labtether");
    std::fs::create_dir_all(&planted).unwrap();
    std::fs::write(
        planted.join("config.json"),
        r#"{"host":"http://127.0.0.1:1","api_key":"planted"}"#,
    )
    .unwrap();
    let out = run(
        &temp,
        &[("LABTETHER_API_KEY", "key-from-env")],
        &["whoami"],
        "",
    );
    let stderr = String::from_utf8_lossy(&out.stderr);
    assert_eq!(out.status.code(), Some(4), "{stderr}");
    assert!(stderr.contains("hub host not configured"), "{stderr}");
    let _ = std::fs::remove_dir_all(&temp);
}

// Fault: `config set-key` wrote the key into <temp folder>/labtether, a
// folder another local user can make first and read.
#[test]
fn settings_are_not_saved_in_the_shared_temp_folder() {
    let temp = shared_temp("save");
    let out = run(&temp, &[], &["config", "set-key"], "key-to-save\n");
    let stderr = String::from_utf8_lossy(&out.stderr);
    assert_eq!(out.status.code(), Some(4), "{stderr}");
    assert!(
        stderr.contains("set LABTETHER_HOST and LABTETHER_API_KEY"),
        "{stderr}"
    );
    assert!(
        !temp.join("labtether").exists(),
        "a settings folder was made in the temporary folder"
    );
    let _ = std::fs::remove_dir_all(&temp);
}

// Promise: with no home folder, LABTETHER_HOST and LABTETHER_API_KEY are
// enough; no config file is needed.
#[test]
fn environment_settings_work_without_a_home_folder() {
    let temp = shared_temp("env");
    let hub = common::hub(|_| r#"{"data":{"auth_type":"api_key","role":"viewer"}}"#.to_string());
    let host = format!("http://127.0.0.1:{}", hub.port);
    let out = run(
        &temp,
        &[
            ("LABTETHER_HOST", &host),
            ("LABTETHER_API_KEY", "key-from-env"),
        ],
        &["whoami"],
        "",
    );
    assert_eq!(
        out.status.code(),
        Some(0),
        "{}",
        String::from_utf8_lossy(&out.stderr)
    );
    assert_eq!(
        hub.requests.lock().unwrap().as_slice(),
        ["GET /api/v2/whoami"]
    );
    let _ = std::fs::remove_dir_all(&temp);
}
