//! Protects the CLI's config file, which holds the API key, as the Go CLI
//! does: mode 0600 on Unix; on Windows a protected ACL that grants access
//! only to the current user, LocalSystem and Administrators.
#![deny(unsafe_code)]

use std::io;
use std::path::Path;

#[cfg(windows)]
mod windows;

/// Restricts `path` to the current user (Go's `hardenConfigFile`).
pub fn harden_config_file(path: &Path) -> io::Result<()> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))
    }
    #[cfg(windows)]
    {
        windows::protect(path)
    }
}

/// Creates a new file only the current user can read, private from the
/// moment it exists: mode 0600 on Unix; on Windows the protected ACL is set
/// at creation and the file is not shared, so no other handle can be open
/// on it while the key is written. Fails if the file exists.
pub fn create_private_file(path: &Path) -> io::Result<std::fs::File> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(path)
    }
    #[cfg(windows)]
    {
        windows::create_private(path)
    }
}

/// Moves the finished temporary file over the config file (Go's
/// `replaceConfigFile`; on Windows `MoveFileEx` with replace).
pub fn replace_config_file(from: &Path, to: &Path) -> io::Result<()> {
    std::fs::rename(from, to)
}

/// Whether the file is private to the current user: mode with no group or
/// other bits on Unix; on Windows a protected DACL with only the current
/// user, LocalSystem and Administrators (what `harden_config_file` sets).
pub fn file_is_private(path: &Path) -> io::Result<bool> {
    #[cfg(unix)]
    {
        Ok(!permissions_too_broad(&std::fs::metadata(path)?))
    }
    #[cfg(windows)]
    {
        windows::is_private(path)
    }
}

/// Whether `mode` lets anyone but the owner read or write. This is the API
/// key file check; like the Go CLI it refuses nothing on Windows (see
/// `file_is_private` for the Windows ACL check).
pub fn permissions_too_broad(meta: &std::fs::Metadata) -> bool {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        meta.permissions().mode() & 0o077 != 0
    }
    #[cfg(not(unix))]
    {
        let _ = meta;
        false
    }
}
