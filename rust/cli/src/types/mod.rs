//! Typed rows for exactly the fields the tables print, one module per area.
//! Each field holds the Hub's raw JSON value so cells print as the Go CLI's
//! `cellText` does. Generated types from the Hub's OpenAPI document can
//! replace these structs once the document describes the responses.

pub mod device;
pub mod hub;
pub mod inventory;
pub mod monitoring;
pub mod platforms;

use serde_json::Value;

/// A member of an object field, as the Go CLI's `lookupPath` reads
/// "metadata.ip_address": `None` when the field is missing or not an object.
pub fn at<'a>(field: &'a Option<Value>, key: &str) -> Option<&'a Value> {
    field.as_ref()?.get(key)
}
