//! Trusted roots: the system roots plus an optional private CA file, as
//! the Go CLI's `NewWithTLSCAFile`. TLS itself is ureq's rustls with the
//! `ring` provider, which speaks TLS 1.2 and 1.3 only.

use std::io::Read;
use std::sync::Arc;

use rustls::RootCertStore;
use rustls_pki_types::CertificateDer;
use rustls_pki_types::pem::PemObject;
use ureq::tls::{Certificate, RootCerts};

use crate::error::{ClientError, path_error};

/// The largest CA file the CLI reads, as in Go.
pub const MAX_CA_FILE_BYTES: u64 = 1024 * 1024;

/// Reads a CA bundle with the Go CLI's checks and messages and returns the
/// certificates it holds.
fn read_ca_file(path: &str) -> Result<Vec<CertificateDer<'static>>, ClientError> {
    let path = path.trim();
    let fail = ClientError::Config;
    let mut file = std::fs::File::open(path).map_err(|e| {
        fail(format!(
            "read TLS CA file: {}",
            path_error("open", path, &e)
        ))
    })?;
    let meta = file.metadata().map_err(|e| {
        fail(format!(
            "inspect TLS CA file: {}",
            path_error("stat", path, &e)
        ))
    })?;
    if !meta.is_file() {
        return Err(fail("TLS CA file must be a regular file".to_string()));
    }
    let mut data = Vec::new();
    file.by_ref()
        .take(MAX_CA_FILE_BYTES + 1)
        .read_to_end(&mut data)
        .map_err(|e| {
            fail(format!(
                "read TLS CA file: {}",
                path_error("read", path, &e)
            ))
        })?;
    if data.len() as u64 > MAX_CA_FILE_BYTES {
        return Err(fail(format!(
            "TLS CA file exceeds {MAX_CA_FILE_BYTES} byte limit"
        )));
    }
    Ok(CertificateDer::pem_slice_iter(&data)
        .filter_map(Result::ok)
        .collect())
}

fn usable(certs: &[CertificateDer<'static>]) -> Vec<CertificateDer<'static>> {
    let mut store = RootCertStore::empty();
    certs
        .iter()
        .filter(|cert| store.add((*cert).clone()).is_ok())
        .cloned()
        .collect()
}

/// Checks a CA file the way `config set-ca` does, without changing any
/// system trust.
pub fn validate_ca_file(path: &str) -> Result<(), ClientError> {
    if usable(&read_ca_file(path)?).is_empty() {
        return Err(ClientError::Config(
            "TLS CA file contains no valid PEM certificates".to_string(),
        ));
    }
    Ok(())
}

/// The trusted roots for one CLI run: the system roots, then the CA
/// file's certificates. ureq takes one root list, so both are combined
/// here. A CA file with no usable certificate is refused.
pub fn root_certs(ca_file: Option<&str>) -> Result<RootCerts, ClientError> {
    let extra = match ca_file {
        Some(path) => {
            let certs = usable(&read_ca_file(path)?);
            if certs.is_empty() {
                return Err(ClientError::Config(
                    "TLS CA file contains no valid PEM certificates".to_string(),
                ));
            }
            certs
        }
        None => Vec::new(),
    };
    let system = usable(&rustls_native_certs::load_native_certs().certs);
    let all: Vec<Certificate<'static>> = system
        .iter()
        .chain(extra.iter())
        .map(|der| Certificate::from_der(der.as_ref()).to_owned())
        .collect();
    Ok(RootCerts::Specific(Arc::new(all)))
}
