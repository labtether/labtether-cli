//! Go's `path/filepath` behaviour for the config path and `config set-ca`.

use std::path::PathBuf;

/// Go's `filepath.Clean` for slash-separated paths.
pub fn clean(path: &str) -> String {
    if path.is_empty() {
        return ".".to_string();
    }
    let rooted = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.last().is_some_and(|p| *p != "..") {
                    parts.pop();
                } else if !rooted {
                    parts.push("..");
                }
            }
            other => parts.push(other),
        }
    }
    let joined = parts.join("/");
    match (rooted, joined.is_empty()) {
        (true, _) => format!("/{joined}"),
        (false, true) => ".".to_string(),
        (false, false) => joined,
    }
}

/// Go's `filepath.Join`.
pub fn join(parts: &[&str]) -> String {
    let kept: Vec<&str> = parts.iter().copied().filter(|p| !p.is_empty()).collect();
    if kept.is_empty() {
        return String::new();
    }
    if cfg!(windows) {
        return kept
            .iter()
            .collect::<PathBuf>()
            .to_string_lossy()
            .into_owned();
    }
    clean(&kept.join("/"))
}

/// Go's `filepath.Abs`.
pub fn absolute(path: &str) -> std::io::Result<String> {
    if cfg!(windows) {
        return std::path::absolute(path).map(|p| p.to_string_lossy().into_owned());
    }
    if path.starts_with('/') {
        return Ok(clean(path));
    }
    let cwd = std::env::current_dir()?;
    Ok(join(&[&cwd.to_string_lossy(), path]))
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;

    // Fault: `config set-ca ../ca.pem` saved a path with ".." in it, or a
    // home folder with a trailing slash gave a different config path than
    // the Go CLI, so the two binaries read different files.
    #[test]
    fn paths_clean_like_go() {
        assert_eq!(clean("/home/demo/../demo//x/./y/"), "/home/demo/x/y");
        assert_eq!(clean("a/../../b"), "../b");
        assert_eq!(clean("/.."), "/");
        assert_eq!(
            join(&["/home/demo/", ".config", "labtether"]),
            "/home/demo/.config/labtether"
        );
    }
}
