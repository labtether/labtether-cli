//! Helpers shared by the Hub commands: requests, list decoding into typed
//! rows, and printing. Where a Hub answer does not have the expected shape,
//! the message says so in plain words (the Go CLI prints Go's decoder
//! text); the exit code is 1 in both.

use serde::de::DeserializeOwned;
use serde_json::Value;

use super::Ctx;
use crate::errors::CliError;
use crate::output::table::{Column, render};

/// A plain name for a JSON value's kind.
pub fn kind(value: &Value) -> &'static str {
    match value {
        Value::Null => "null",
        Value::Bool(_) => "true or false",
        Value::Number(_) => "a number",
        Value::String(_) => "text",
        Value::Array(_) => "a list",
        Value::Object(_) => "an object",
    }
}

fn decode_failed(detail: String) -> CliError {
    CliError::Failed(format!("decode response data: {detail}"))
}

/// The Go CLI's `decodeRows`: a list of objects; null is an empty list and
/// a null item is an empty row.
pub fn decode_rows(raw: &Value) -> Result<Vec<&Value>, String> {
    match raw {
        Value::Null => Ok(Vec::new()),
        Value::Array(items) => {
            if let Some((i, item)) = items
                .iter()
                .enumerate()
                .find(|(_, v)| !matches!(v, Value::Object(_) | Value::Null))
            {
                return Err(format!("item {i} is {}, not an object", kind(item)));
            }
            Ok(items.iter().collect())
        }
        other => Err(format!("the hub sent {}, not a list", kind(other))),
    }
}

/// Reads rows into their typed struct; a null row is an empty one.
pub fn typed<T: DeserializeOwned + Default>(rows: Vec<&Value>) -> Result<Vec<T>, String> {
    rows.into_iter()
        .enumerate()
        .map(|(i, row)| match row {
            Value::Null => Ok(T::default()),
            row => T::deserialize(row).map_err(|e| format!("item {i}: {e}")),
        })
        .collect()
}

/// The payload must be an object (or null), as the Go CLI's
/// `decodeResponseData` into a map requires.
pub fn decode_object(data: &Value) -> Result<(), CliError> {
    match data {
        Value::Object(_) | Value::Null => Ok(()),
        other => Err(decode_failed(format!(
            "the hub sent {}, not an object",
            kind(other)
        ))),
    }
}

/// The Go CLI's `decodeTableRows`: the named list from the payload.
pub fn decode_list<'a>(data: &'a Value, field: &str) -> Result<Vec<&'a Value>, CliError> {
    decode_object(data)?;
    let raw = data
        .get(field)
        .ok_or_else(|| decode_failed(format!("missing {field}")))?;
    decode_rows(raw).map_err(|e| decode_failed(format!("{field}: {e}")))
}

/// The named list as typed rows.
pub fn rows<T: DeserializeOwned + Default>(data: &Value, field: &str) -> Result<Vec<T>, CliError> {
    typed(decode_list(data, field)?).map_err(|e| decode_failed(format!("{field}: {e}")))
}

/// A payload Go decodes into `map[string]any` (whoami, docker logs).
pub fn check_any_map(data: &Value) -> Result<(), String> {
    match data {
        Value::Object(_) | Value::Null => Ok(()),
        other => Err(format!("the hub sent {}, not an object", kind(other))),
    }
}

impl Ctx<'_> {
    /// `GET path` through a client built from the settings.
    pub fn get(&mut self, path: &str) -> Result<Value, CliError> {
        let client = self.client()?;
        Ok(client.get(path)?)
    }

    /// Prints the Hub's payload as JSON (the Go CLI's
    /// `printJSON(json.RawMessage(resp.Data))`).
    pub fn print_data(&mut self, data: &Value) -> Result<(), CliError> {
        self.out.print_json(data);
        Ok(())
    }

    /// The Go CLI's `printList`: JSON prints the payload; otherwise the
    /// named list as a table of typed rows.
    pub fn print_list<T: DeserializeOwned + Default>(
        &mut self,
        data: &Value,
        field: &str,
        columns: &[Column<T>],
    ) -> Result<(), CliError> {
        if self.json {
            return self.print_data(data);
        }
        let rows: Vec<T> = rows(data, field)?;
        self.out.print(&render(&rows, columns));
        Ok(())
    }

    /// GET a path, then print its payload as JSON.
    pub fn get_and_print(&mut self, path: &str) -> Result<(), CliError> {
        let data = self.get(path)?;
        self.print_data(&data)
    }

    /// A positional argument, path-escaped as Go's `url.PathEscape`.
    pub fn arg(&self, index: usize) -> String {
        labtether_cli_client::escape::path_escape(&self.args[index])
    }

    /// A positional argument, query-escaped as Go's `url.QueryEscape`.
    pub fn query_arg(&self, index: usize) -> String {
        labtether_cli_client::escape::query_escape(&self.args[index])
    }
}
