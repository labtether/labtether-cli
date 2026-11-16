//! The LabTether command-line tool in Rust. It copies the Go CLI's
//! commands, flags, settings, output and exit codes; commands that change
//! something are refused for now.
#![forbid(unsafe_code)]

pub mod app;
pub mod cobra;
pub mod commands;
pub mod errors;
pub mod output;
pub mod paths;
pub mod settings;
pub mod types;
