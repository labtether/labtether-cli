#![allow(dead_code)]
//! A recording stand-in Hub and a way to run the CLI in-process.

use std::io::{BufRead, BufReader, Read, Write};
use std::net::TcpListener;
use std::sync::{Arc, Mutex};
use std::time::Duration;

use labtether_cli_client::Secret;
use labtether_cli_rs::app::{Input, execute};
use labtether_cli_rs::output::Out;
use labtether_cli_rs::settings::Env;

/// A Hub that answers every request with `{"data": <data>}` and records
/// "METHOD target" for each request.
pub struct Hub {
    pub port: u16,
    pub requests: Arc<Mutex<Vec<String>>>,
}

pub fn hub(answer: impl Fn(&str) -> String + Send + 'static) -> Hub {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    let requests = Arc::new(Mutex::new(Vec::new()));
    let record = requests.clone();
    std::thread::spawn(move || {
        for stream in listener.incoming() {
            let Ok(mut stream) = stream else { break };
            let mut reader = BufReader::new(stream.try_clone().unwrap());
            let mut line = String::new();
            let _ = reader.read_line(&mut line);
            let mut length = 0usize;
            loop {
                let mut header = String::new();
                if reader.read_line(&mut header).unwrap_or(0) == 0 || header == "\r\n" {
                    break;
                }
                if let Some(v) = header.to_ascii_lowercase().strip_prefix("content-length:") {
                    length = v.trim().parse().unwrap_or(0);
                }
            }
            let mut body = vec![0; length];
            let _ = reader.read_exact(&mut body);
            let target: Vec<&str> = line.split(' ').take(2).collect();
            let request = target.join(" ");
            record.lock().unwrap().push(request.clone());
            let body = answer(&request);
            let _ = stream.write_all(
                format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n{body}", body.len())
                    .as_bytes(),
            );
        }
    });
    Hub { port, requests }
}

pub struct NoInput;

impl Input for NoInput {
    fn is_terminal(&self) -> bool {
        false
    }
    fn read_limited(&mut self, _limit: u64) -> std::io::Result<Vec<u8>> {
        Ok(Vec::new())
    }
    fn read_password(&mut self) -> std::io::Result<String> {
        Ok(String::new())
    }
}

/// A private, empty home folder so no test reads a real config.
pub fn temp_home(name: &str) -> String {
    let dir = std::env::temp_dir().join(format!("ltcli-it-{name}-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).unwrap();
    dir.to_string_lossy().into_owned()
}

pub struct Run {
    pub code: i32,
    pub stdout: String,
    pub stderr: String,
}

/// Runs the CLI with the stand-in Hub's address and a test key in the
/// environment.
pub fn run_against(port: u16, home: &str, args: &[String]) -> Run {
    let env = Env {
        host: Some(format!("http://127.0.0.1:{port}")),
        api_key: Some(Secret::new("test-key")),
        home: Some(home.to_string()),
        ..Env::default()
    };
    let mut out = Out::default();
    let code = execute(
        args,
        &env,
        &mut NoInput,
        &mut out,
        Some(Duration::from_secs(10)),
    );
    Run {
        code,
        stdout: String::from_utf8_lossy(&out.stdout).into_owned(),
        stderr: String::from_utf8_lossy(&out.stderr).into_owned(),
    }
}
