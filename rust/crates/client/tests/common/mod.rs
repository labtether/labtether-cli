#![allow(dead_code)]
//! A scripted stand-in Hub on a loopback port: it records each request
//! head and answers with bytes from a closure.

use std::io::{BufRead, BufReader, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::{Arc, Mutex};
use std::thread;

pub type Answer = Box<dyn Fn(&str, u16) -> Vec<u8> + Send + Sync>;

pub struct StandIn {
    pub port: u16,
    pub requests: Arc<Mutex<Vec<String>>>,
}

pub fn read_head(stream: &mut TcpStream) -> String {
    let mut reader = BufReader::new(stream.try_clone().unwrap());
    let mut head = String::new();
    loop {
        let mut line = String::new();
        if reader.read_line(&mut line).unwrap_or(0) == 0 || line == "\r\n" {
            break;
        }
        head.push_str(&line);
    }
    head
}

/// Serves every connection with `answer(request_line, own_port)`.
pub fn serve(answer: Answer) -> StandIn {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    let requests = Arc::new(Mutex::new(Vec::new()));
    let record = requests.clone();
    thread::spawn(move || {
        for stream in listener.incoming() {
            let Ok(mut stream) = stream else { break };
            let head = read_head(&mut stream);
            let line = head.lines().next().unwrap_or_default().to_string();
            record.lock().unwrap().push(head);
            let _ = stream.write_all(&answer(&line, port));
        }
    });
    StandIn { port, requests }
}

pub fn ok_json(body: &str) -> Vec<u8> {
    format!(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\n\r\n{body}",
        body.len()
    )
    .into_bytes()
}

pub fn redirect(location: &str) -> Vec<u8> {
    format!("HTTP/1.1 302 Found\r\nLocation: {location}\r\nContent-Length: 0\r\n\r\n").into_bytes()
}
