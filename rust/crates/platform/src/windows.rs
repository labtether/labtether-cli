//! The Windows config file ACL, as the Go CLI's `config_security_windows.go`:
//! a protected DACL (no inherited entries) granting GENERIC_ALL to the
//! current user, LocalSystem and the Administrators group, and a read-back
//! that checks a file still has only those entries.
//!
//! This is the only file in the workspace with `unsafe` code. Every block
//! calls a documented Win32 function with buffers this file owns and sizes,
//! and frees what Windows allocates before returning.
#![allow(unsafe_code)]

use std::fs::File;
use std::io;
use std::os::windows::ffi::OsStrExt;
use std::os::windows::io::{FromRawHandle, RawHandle};
use std::path::Path;
use std::ptr::{null, null_mut};

use windows_sys::Win32::Foundation::{
    CloseHandle, ERROR_SUCCESS, GENERIC_ALL, GENERIC_READ, GENERIC_WRITE, HANDLE,
    INVALID_HANDLE_VALUE, LocalFree,
};
use windows_sys::Win32::Security::Authorization::{
    EXPLICIT_ACCESS_W, GRANT_ACCESS, GetNamedSecurityInfoW, NO_MULTIPLE_TRUSTEE, SE_FILE_OBJECT,
    SetEntriesInAclW, SetNamedSecurityInfoW, TRUSTEE_IS_GROUP, TRUSTEE_IS_SID, TRUSTEE_IS_USER,
    TRUSTEE_W,
};
use windows_sys::Win32::Security::{
    ACCESS_ALLOWED_ACE, ACE_HEADER, ACL, ACL_SIZE_INFORMATION, AclSizeInformation,
    CreateWellKnownSid, DACL_SECURITY_INFORMATION, EqualSid, GetAce, GetAclInformation,
    GetSecurityDescriptorControl, GetTokenInformation, InitializeSecurityDescriptor,
    NO_INHERITANCE, PROTECTED_DACL_SECURITY_INFORMATION, PSECURITY_DESCRIPTOR, SE_DACL_PROTECTED,
    SECURITY_ATTRIBUTES, SECURITY_DESCRIPTOR, SetSecurityDescriptorControl,
    SetSecurityDescriptorDacl, TOKEN_QUERY, TOKEN_USER, TokenUser, WinBuiltinAdministratorsSid,
    WinLocalSystemSid,
};
use windows_sys::Win32::Storage::FileSystem::{CREATE_NEW, CreateFileW, FILE_ATTRIBUTE_NORMAL};
use windows_sys::Win32::System::SystemServices::SECURITY_DESCRIPTOR_REVISION;
use windows_sys::Win32::System::Threading::{GetCurrentProcess, OpenProcessToken};

/// SECURITY_MAX_SID_SIZE: no SID is longer.
const SID_BYTES: usize = 68;
/// ACCESS_ALLOWED_ACE_TYPE.
const ALLOWED_ACE: u8 = 0;
/// ACCESS_DENIED_ACE_TYPE.
const DENIED_ACE: u8 = 1;

fn last_error() -> io::Error {
    io::Error::last_os_error()
}

fn wide(path: &Path) -> Vec<u16> {
    path.as_os_str()
        .encode_wide()
        .chain(std::iter::once(0))
        .collect()
}

/// The current user's SID, copied out of the process token.
fn current_user_sid() -> io::Result<Vec<u8>> {
    let mut token: HANDLE = null_mut();
    // SAFETY: GetCurrentProcess returns a pseudo handle that needs no
    // closing; `token` is a valid out pointer and receives a real handle,
    // which is closed at the end of this function on every path.
    if unsafe { OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &mut token) } == 0 {
        return Err(last_error());
    }
    let mut needed = 0u32;
    // SAFETY: a size query: a null buffer of length 0 is allowed, and
    // `needed` is a valid out pointer. The call fails by design here.
    unsafe { GetTokenInformation(token, TokenUser, null_mut(), 0, &mut needed) };
    let mut buffer = vec![0u64; (needed as usize).div_ceil(8).max(1)];
    // SAFETY: `buffer` is at least `needed` bytes and 8-byte aligned, as
    // TOKEN_USER needs; Windows writes at most `needed` bytes.
    let ok = unsafe {
        GetTokenInformation(
            token,
            TokenUser,
            buffer.as_mut_ptr().cast(),
            needed,
            &mut needed,
        )
    };
    let result = if ok == 0 {
        Err(last_error())
    } else {
        // SAFETY: on success the buffer starts with a TOKEN_USER whose SID
        // pointer points into the same buffer; the copy is clamped to the
        // bytes Windows wrote after that pointer and to the largest SID.
        let user = unsafe { &*(buffer.as_ptr() as *const TOKEN_USER) };
        let sid = user.User.Sid as *const u8;
        let offset = (sid as usize).saturating_sub(buffer.as_ptr() as usize);
        let available = (needed as usize).saturating_sub(offset).min(SID_BYTES);
        // SAFETY: `sid..sid + available` lies inside `buffer` (see above).
        Ok(unsafe { std::slice::from_raw_parts(sid, available) }.to_vec())
    };
    // SAFETY: `token` was opened above and is closed once.
    unsafe { CloseHandle(token) };
    result
}

fn well_known_sid(kind: i32) -> io::Result<Vec<u8>> {
    let mut sid = vec![0u8; SID_BYTES];
    let mut size = SID_BYTES as u32;
    // SAFETY: `sid` has room for SECURITY_MAX_SID_SIZE bytes and `size`
    // says so; a null domain SID is allowed for these well-known SIDs.
    if unsafe { CreateWellKnownSid(kind, null_mut(), sid.as_mut_ptr().cast(), &mut size) } == 0 {
        return Err(last_error());
    }
    sid.truncate(size as usize);
    Ok(sid)
}

/// The three SIDs the config file may grant: user, LocalSystem, Administrators.
fn allowed_sids() -> io::Result<[Vec<u8>; 3]> {
    Ok([
        current_user_sid()?,
        well_known_sid(WinLocalSystemSid)?,
        well_known_sid(WinBuiltinAdministratorsSid)?,
    ])
}

fn entry(sid: &mut [u8], trustee_type: i32) -> EXPLICIT_ACCESS_W {
    EXPLICIT_ACCESS_W {
        grfAccessPermissions: GENERIC_ALL,
        grfAccessMode: GRANT_ACCESS,
        grfInheritance: NO_INHERITANCE,
        Trustee: TRUSTEE_W {
            pMultipleTrustee: null_mut(),
            MultipleTrusteeOperation: NO_MULTIPLE_TRUSTEE,
            TrusteeForm: TRUSTEE_IS_SID,
            TrusteeType: trustee_type,
            ptstrName: sid.as_mut_ptr().cast(),
        },
    }
}

/// The protected three-entry ACL, freed when dropped.
struct PrivateAcl(*mut ACL);

impl Drop for PrivateAcl {
    fn drop(&mut self) {
        // SAFETY: the pointer came from SetEntriesInAclW and is freed once.
        unsafe { LocalFree(self.0.cast()) };
    }
}

/// Builds the ACL that grants GENERIC_ALL to the user, LocalSystem and
/// Administrators, and nothing else.
fn private_acl() -> io::Result<PrivateAcl> {
    let [mut user, mut system, mut admins] = allowed_sids()?;
    let entries = [
        entry(&mut user, TRUSTEE_IS_USER),
        entry(&mut system, TRUSTEE_IS_USER),
        entry(&mut admins, TRUSTEE_IS_GROUP),
    ];
    let mut acl: *mut ACL = null_mut();
    // SAFETY: `entries` and the SID buffers they point to live until the end
    // of this function; no old ACL is merged (null); `acl` receives a
    // LocalAlloc'd ACL that `PrivateAcl` frees. SetEntriesInAclW copies the
    // SIDs into the ACL, so it does not borrow the buffers.
    let status =
        unsafe { SetEntriesInAclW(entries.len() as u32, entries.as_ptr(), null(), &mut acl) };
    if status != ERROR_SUCCESS {
        return Err(io::Error::from_raw_os_error(status as i32));
    }
    Ok(PrivateAcl(acl))
}

/// Replaces the file's DACL with the protected three-entry list.
pub fn protect(path: &Path) -> io::Result<()> {
    let acl = private_acl()?;
    let name = wide(path);
    // SAFETY: `name` is NUL-terminated and outlives the call; owner, group
    // and SACL are not changed (null); `acl` is the ACL built above.
    let status = unsafe {
        SetNamedSecurityInfoW(
            name.as_ptr(),
            SE_FILE_OBJECT,
            DACL_SECURITY_INFORMATION | PROTECTED_DACL_SECURITY_INFORMATION,
            null_mut(),
            null_mut(),
            acl.0,
            null(),
        )
    };
    if status != ERROR_SUCCESS {
        return Err(io::Error::from_raw_os_error(status as i32));
    }
    Ok(())
}

/// Creates a new file with the protected ACL set at creation, so it never
/// has the folder's inherited access, and with no sharing, so no other
/// handle can be opened on it while this one is open.
pub fn create_private(path: &Path) -> io::Result<File> {
    let acl = private_acl()?;
    let mut descriptor = SECURITY_DESCRIPTOR::default();
    let descriptor_ptr: PSECURITY_DESCRIPTOR = (&mut descriptor as *mut SECURITY_DESCRIPTOR).cast();
    // SAFETY: `descriptor` is a SECURITY_DESCRIPTOR this function owns and
    // that outlives every use below.
    if unsafe { InitializeSecurityDescriptor(descriptor_ptr, SECURITY_DESCRIPTOR_REVISION) } == 0 {
        return Err(last_error());
    }
    // SAFETY: the descriptor is initialised; `acl` lives until the end of
    // this function, after CreateFileW has copied it to the new file.
    if unsafe { SetSecurityDescriptorDacl(descriptor_ptr, 1, acl.0, 0) } == 0 {
        return Err(last_error());
    }
    // SAFETY: the descriptor is initialised; only the protected bit is set.
    if unsafe { SetSecurityDescriptorControl(descriptor_ptr, SE_DACL_PROTECTED, SE_DACL_PROTECTED) }
        == 0
    {
        return Err(last_error());
    }
    let attributes = SECURITY_ATTRIBUTES {
        nLength: size_of::<SECURITY_ATTRIBUTES>() as u32,
        lpSecurityDescriptor: descriptor_ptr,
        bInheritHandle: 0,
    };
    let name = wide(path);
    // SAFETY: `name` is NUL-terminated; `attributes` and the descriptor and
    // ACL it points to live until the call returns; share mode 0 and
    // CREATE_NEW; no template file.
    let handle = unsafe {
        CreateFileW(
            name.as_ptr(),
            GENERIC_READ | GENERIC_WRITE,
            0,
            &attributes,
            CREATE_NEW,
            FILE_ATTRIBUTE_NORMAL,
            null_mut(),
        )
    };
    if handle == INVALID_HANDLE_VALUE {
        return Err(last_error());
    }
    // SAFETY: `handle` is a valid file handle that nothing else owns; the
    // File closes it.
    Ok(unsafe { File::from_raw_handle(handle as RawHandle) })
}

/// Whether the file's DACL is protected (inherits nothing) and allows only
/// the current user, LocalSystem and Administrators.
pub fn is_private(path: &Path) -> io::Result<bool> {
    let allowed = allowed_sids()?;
    let name = wide(path);
    let mut dacl: *mut ACL = null_mut();
    let mut descriptor: PSECURITY_DESCRIPTOR = null_mut();
    // SAFETY: `name` is NUL-terminated; only the DACL is asked for, so the
    // other out pointers may be null; `dacl` points into `descriptor`,
    // which Windows allocates and this function frees below.
    let status = unsafe {
        GetNamedSecurityInfoW(
            name.as_ptr(),
            SE_FILE_OBJECT,
            DACL_SECURITY_INFORMATION,
            null_mut(),
            null_mut(),
            &mut dacl,
            null_mut(),
            &mut descriptor,
        )
    };
    if status != ERROR_SUCCESS {
        return Err(io::Error::from_raw_os_error(status as i32));
    }
    let result = check_dacl(descriptor, dacl, &allowed);
    // SAFETY: `descriptor` came from GetNamedSecurityInfoW and is freed once;
    // `dacl` is not used after this.
    unsafe { LocalFree(descriptor) };
    result
}

fn check_dacl(
    descriptor: PSECURITY_DESCRIPTOR,
    dacl: *mut ACL,
    allowed: &[Vec<u8>; 3],
) -> io::Result<bool> {
    // A null DACL grants everyone everything.
    if dacl.is_null() {
        return Ok(false);
    }
    let mut control = 0u16;
    let mut revision = 0u32;
    // SAFETY: `descriptor` is the live descriptor from GetNamedSecurityInfoW;
    // both out pointers are valid.
    if unsafe { GetSecurityDescriptorControl(descriptor, &mut control, &mut revision) } == 0 {
        return Err(last_error());
    }
    if control & SE_DACL_PROTECTED == 0 {
        return Ok(false);
    }
    let mut info = ACL_SIZE_INFORMATION {
        AceCount: 0,
        AclBytesInUse: 0,
        AclBytesFree: 0,
    };
    // SAFETY: `info` is an ACL_SIZE_INFORMATION and the length passed is its
    // size; `dacl` is non-null and lives inside the descriptor.
    let ok = unsafe {
        GetAclInformation(
            dacl,
            (&mut info as *mut ACL_SIZE_INFORMATION).cast(),
            size_of::<ACL_SIZE_INFORMATION>() as u32,
            AclSizeInformation,
        )
    };
    if ok == 0 {
        return Err(last_error());
    }
    for index in 0..info.AceCount {
        let mut ace: *mut core::ffi::c_void = null_mut();
        // SAFETY: `index` is below the ACL's own ACE count; `ace` receives a
        // pointer into the ACL, valid while the descriptor lives.
        if unsafe { GetAce(dacl, index, &mut ace) } == 0 {
            return Err(last_error());
        }
        // SAFETY: every ACE starts with an ACE_HEADER.
        let header = unsafe { &*(ace as *const ACE_HEADER) };
        // A deny entry cannot widen access. Any other kind of entry (an
        // object, callback or conditional allow) can, and `protect` never
        // writes one, so a file that has one is not private.
        if header.AceType == DENIED_ACE {
            continue;
        }
        if header.AceType != ALLOWED_ACE {
            return Ok(false);
        }
        // SAFETY: an ACCESS_ALLOWED_ACE's SID starts at its SidStart field
        // and lies within the ACE, which Windows has validated.
        let sid = unsafe { &(*(ace as *const ACCESS_ALLOWED_ACE)).SidStart } as *const u32;
        // SAFETY: both SIDs are valid: one from the ACL, one from our buffer.
        let known = allowed
            .iter()
            .any(|ok| unsafe { EqualSid(sid as _, ok.as_ptr() as _) } != 0);
        if !known {
            return Ok(false);
        }
    }
    Ok(true)
}
