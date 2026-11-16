use super::*;

/// A private folder for one test, used as the home folder.
fn temp_home(name: &str) -> String {
    let dir = std::env::temp_dir().join(format!("ltcli-test-{name}-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    dir.to_string_lossy().into_owned()
}

fn env_with_home(home: &str) -> Env {
    Env {
        home: Some(home.to_string()),
        ..Env::default()
    }
}

fn write_config(env: &Env, text: &str) {
    std::fs::create_dir_all(env.config_dir()).unwrap();
    std::fs::write(env.config_path(), text).unwrap();
}

#[cfg(unix)]
fn write_mode(path: &str, text: &str, mode: u32) {
    use std::os::unix::fs::PermissionsExt;
    std::fs::write(path, text).unwrap();
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode)).unwrap();
}

// Fault: precedence differed from the Go CLI and the README (flag >
// environment > config file), so the same command reached a different Hub.
#[test]
fn flags_beat_environment_beat_config() {
    let home = temp_home("precedence");
    let mut env = env_with_home(&home);
    write_config(
        &env,
        r#"{"host":"https://config.example","api_key":"config-key","tls_ca_file":"/config/ca.pem"}"#,
    );
    let resolved = resolve(&env, &Flags::default()).unwrap();
    assert_eq!(
        (
            resolved.host.as_str(),
            resolved.api_key.expose(),
            resolved.ca_file.as_str()
        ),
        ("https://config.example", "config-key", "/config/ca.pem")
    );
    env.host = Some("https://env.example".into());
    env.api_key = Some(Secret::new("env-key"));
    env.ca_file = Some("/env/ca.pem".into());
    let resolved = resolve(&env, &Flags::default()).unwrap();
    assert_eq!(
        (
            resolved.host.as_str(),
            resolved.api_key.expose(),
            resolved.ca_file.as_str()
        ),
        ("https://env.example", "env-key", "/env/ca.pem")
    );
    let flags = Flags {
        host: "https://flag.example".into(),
        ca_file: "/flag/ca.pem".into(),
        ..Flags::default()
    };
    let resolved = resolve(&env, &flags).unwrap();
    assert_eq!(
        (resolved.host.as_str(), resolved.ca_file.as_str()),
        ("https://flag.example", "/flag/ca.pem")
    );
    assert_eq!(resolved.api_key.expose(), "env-key");
}

// Fault: the API key given on the command line was used, so it showed up
// in shell history and process lists. The Go CLI refuses it, exit 4.
#[test]
fn api_key_argument_is_refused() {
    let home = temp_home("argkey");
    let env = Env {
        host: Some("https://env.example".into()),
        api_key: Some(Secret::new("env-key")),
        ..env_with_home(&home)
    };
    let flags = Flags {
        api_key: "flag-key".into(),
        ..Flags::default()
    };
    let err = resolve(&env, &flags).unwrap_err();
    assert_eq!(
        err,
        CliError::Config(
            "refusing API key in command-line arguments; use --api-key-file or LABTETHER_API_KEY"
                .into()
        )
    );
    assert_eq!(err.exit_code(), 4);
}

// Fault: a key file readable by other users was accepted, or the key file
// did not win over the environment.
#[cfg(unix)]
#[test]
fn key_file_must_be_private_and_wins_over_environment() {
    let home = temp_home("keyfile");
    let key_path = format!("{home}/key");
    let env = Env {
        host: Some("https://env.example".into()),
        api_key: Some(Secret::new("env-key")),
        ..env_with_home(&home)
    };
    write_mode(&key_path, "file-key\n", 0o644);
    let flags = Flags {
        api_key_file: key_path.clone(),
        ..Flags::default()
    };
    assert_eq!(
        resolve(&env, &flags).unwrap_err(),
        CliError::Config("API key file permissions are too broad; require mode 0600".into())
    );
    write_mode(&key_path, "  file-key\n", 0o600);
    assert_eq!(resolve(&env, &flags).unwrap().api_key.expose(), "file-key");
    write_mode(&key_path, " \n", 0o600);
    assert_eq!(
        resolve(&env, &flags).unwrap_err(),
        CliError::Config("API key file is empty".into())
    );
}

// Fault: a symlinked config was followed (it could point at another
// user's file), or a config readable by others stayed that way.
#[cfg(unix)]
#[test]
fn config_symlink_is_refused_and_config_is_made_private() {
    use std::os::unix::fs::PermissionsExt;
    let home = temp_home("symlink");
    let env = env_with_home(&home);
    std::fs::create_dir_all(env.config_dir()).unwrap();
    let target = format!("{home}/elsewhere.json");
    write_mode(&target, r#"{"host":"https://x.example"}"#, 0o600);
    std::os::unix::fs::symlink(&target, env.config_path()).unwrap();
    let (_, err) = load_config(&env);
    assert_eq!(
        err,
        Some(format!(
            "config {} must be a regular file, not a symlink or device",
            env.config_path()
        ))
    );
    std::fs::remove_file(env.config_path()).unwrap();
    write_mode(&env.config_path(), r#"{"host":"https://x.example"}"#, 0o644);
    let (config, err) = load_config(&env);
    assert_eq!((config.host.as_str(), err), ("https://x.example", None));
    let mode = std::fs::metadata(env.config_path())
        .unwrap()
        .permissions()
        .mode()
        & 0o777;
    assert_eq!(mode, 0o600);
}

// Fault: a broken config file blocked commands even when the environment
// gave a complete host and key, or a broken one was used without a word.
#[test]
fn broken_config_and_environment_follow_go() {
    let home = temp_home("broken");
    let mut env = env_with_home(&home);
    write_config(&env, r#"{"host":"#);
    let err = resolve(&env, &Flags::default()).unwrap_err();
    assert!(
        matches!(&err, CliError::Config(m) if m.starts_with(&format!("parse config {}: the file is not a valid config (", env.config_path()))),
        "{err:?}"
    );
    env.host = Some("https://env.example".into());
    env.api_key = Some(Secret::new("env-key"));
    assert_eq!(
        resolve(&env, &Flags::default()).unwrap().host,
        "https://env.example"
    );
    // A field of the wrong type makes the whole file unreadable here (the
    // Go CLI keeps the fields that decoded); the environment still wins.
    write_config(&env, r#"{"host":"https://config.example","api_key":5}"#);
    assert_eq!(
        resolve(&env, &Flags::default()).unwrap().host,
        "https://env.example"
    );
    env.host = None;
    let err = resolve(&env, &Flags::default()).unwrap_err();
    assert!(
        matches!(&err, CliError::Config(m) if m.starts_with(&format!("parse config {}: the file is not a valid config (", env.config_path()))),
        "{err:?}"
    );
}

#[test]
fn missing_settings_name_the_fix() {
    let home = temp_home("missing");
    let mut env = env_with_home(&home);
    assert_eq!(
        resolve(&env, &Flags::default()).unwrap_err(),
        CliError::Config(
            "hub host not configured; run: labtether-cli config set-host <url>".into()
        )
    );
    env.host = Some("https://env.example".into());
    assert_eq!(
        resolve(&env, &Flags::default()).unwrap_err(),
        CliError::Config("API key not configured; run: labtether-cli config set-key".into())
    );
}

// Fault: the saved config was readable by other users, or its layout
// differed from the Go CLI's so the Go CLI could not read it back.
#[cfg(unix)]
#[test]
fn saved_config_is_private_and_go_shaped() {
    use std::os::unix::fs::PermissionsExt;
    let home = temp_home("save");
    let env = env_with_home(&home);
    let config = Config {
        host: "https://nas01".into(),
        api_key: Some(Secret::new("k<1>")),
        tls_ca_file: String::new(),
    };
    save_config(&env, &config).unwrap();
    let text = std::fs::read_to_string(env.config_path()).unwrap();
    assert_eq!(
        text,
        "{\n  \"host\": \"https://nas01\",\n  \"api_key\": \"k\\u003c1\\u003e\"\n}"
    );
    assert_eq!(
        std::fs::metadata(env.config_path())
            .unwrap()
            .permissions()
            .mode()
            & 0o777,
        0o600
    );
}

// Fault: the API key reached a debug print or error text.
#[test]
fn secrets_never_print() {
    let resolved = Resolved {
        host: "https://h".into(),
        api_key: Secret::new("lt_do_not_print"),
        ca_file: String::new(),
    };
    assert!(!format!("{resolved:?}").contains("lt_do_not_print"));
}
