//! The LabTether command-line tool in Rust. It copies the Go CLI's
//! commands, flags, settings, requests, output and exit codes.
#![forbid(unsafe_code)]

pub mod app;
pub mod cobra;
pub mod commands;
pub mod errors;
pub mod output;
pub mod paths;
pub mod settings;
pub mod types;
