//! The config file protection on the real file system: what
//! `harden_config_file` leaves, read back by `file_is_private` and, on
//! Windows, by `icacls`.

use std::path::PathBuf;

use labtether_cli_platform::{file_is_private, harden_config_file};

fn temp_file(name: &str) -> PathBuf {
    let path = std::env::temp_dir().join(format!("ltcli-acl-{name}-{}", std::process::id()));
    std::fs::write(&path, "{}").unwrap();
    path
}

// Fault: the config file (it holds the API key) stayed readable by other
// users after the CLI saved it, or a widened file was reported as private.
#[test]
fn hardened_file_is_private_and_widening_is_caught() {
    let path = temp_file("harden");
    harden_config_file(&path).unwrap();
    assert!(
        file_is_private(&path).unwrap(),
        "hardened file is not private"
    );
    #[cfg(windows)]
    {
        let out = std::process::Command::new("icacls")
            .arg(&path)
            .output()
            .unwrap();
        let text = String::from_utf8_lossy(&out.stdout).to_string();
        println!("icacls after harden:\n{text}");
        let user = std::env::var("USERNAME").unwrap().to_lowercase();
        let lines: Vec<String> = text
            .lines()
            .map(|l| l.to_lowercase())
            .filter(|l| l.contains(":("))
            .collect();
        assert_eq!(lines.len(), 3, "{text}");
        assert!(
            lines.iter().any(|l| l.contains(&user) && l.contains("(f)")),
            "{text}"
        );
        assert!(
            lines
                .iter()
                .any(|l| l.contains("system") && l.contains("(f)")),
            "{text}"
        );
        assert!(
            lines
                .iter()
                .any(|l| l.contains("administrators") && l.contains("(f)")),
            "{text}"
        );
        assert!(
            !lines.iter().any(|l| l.contains("(i)")),
            "inherited entries left: {text}"
        );
        let widened = std::process::Command::new("icacls")
            .arg(&path)
            .args(["/grant", "*S-1-1-0:R"])
            .status()
            .unwrap();
        assert!(widened.success());
    }
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        assert_eq!(
            std::fs::metadata(&path).unwrap().permissions().mode() & 0o777,
            0o600
        );
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644)).unwrap();
    }
    assert!(
        !file_is_private(&path).unwrap(),
        "widened file reported as private"
    );
    let _ = std::fs::remove_file(&path);
}

// Fault: the config's temporary copy, which receives the API key, was
// created with the folder's inherited access and protected only after. A
// handle another program opened in that moment keeps the access it was
// opened with and could read the key written next.
#[test]
fn new_config_file_is_private_from_creation() {
    let path = std::env::temp_dir().join(format!("ltcli-acl-new-{}", std::process::id()));
    let _ = std::fs::remove_file(&path);
    let file = labtether_cli_platform::create_private_file(&path).unwrap();
    assert!(
        file_is_private(&path).unwrap(),
        "the new file is not private before anything else happens"
    );
    #[cfg(windows)]
    assert!(
        std::fs::File::open(&path).is_err(),
        "a second handle was opened while the file is being written"
    );
    drop(file);
    let _ = std::fs::remove_file(&path);
}
