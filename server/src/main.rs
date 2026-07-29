mod pages;

use std::fs;
use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::path::PathBuf;
use std::thread;

fn static_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../static")
}

fn content_type(path: &str) -> &'static str {
    if path.ends_with(".css") {
        "text/css; charset=utf-8"
    } else if path.ends_with(".js") {
        "application/javascript; charset=utf-8"
    } else if path.ends_with(".html") {
        "text/html; charset=utf-8"
    } else {
        "application/octet-stream"
    }
}

fn http_response(status: u16, status_text: &str, content_type: &str, body: &[u8]) -> Vec<u8> {
    format!(
        "HTTP/1.1 {} {}\r\nContent-Type: {}\r\nContent-Length: {}\r\nConnection: close\r\nAccess-Control-Allow-Origin: *\r\n\r\n",
        status, status_text, content_type, body.len()
    )
    .into_bytes()
    .into_iter()
    .chain(body.iter().cloned())
    .collect()
}

fn handle_client(mut stream: TcpStream) {
    let mut buf = [0u8; 2048];
    let n = stream.read(&mut buf).unwrap_or(0);
    if n == 0 {
        return;
    }

    let request = String::from_utf8_lossy(&buf[..n]);
    let first_line = request.lines().next().unwrap_or("");
    let raw_path = first_line.split_whitespace().nth(1).unwrap_or("/");
    // Ignora query string (?asn=…, ?token=…) para o match de rotas
    let path = raw_path.split('?').next().unwrap_or("/");

    let response = if path.starts_with("/static/") {
        let file_path = static_dir().join(path.trim_start_matches("/static/"));
        match fs::read(&file_path) {
            Ok(data) => http_response(200, "OK", content_type(path), &data),
            Err(_) => {
                let body = b"404 Not Found";
                http_response(404, "Not Found", "text/plain", body)
            }
        }
    } else {
        let (html, ct) = match path {
            "/" => (pages::render_dashboard(), "text/html; charset=utf-8"),
            "/cdns" => (pages::render_cdns(), "text/html; charset=utf-8"),
            "/peers" => (pages::render_peers(), "text/html; charset=utf-8"),
            "/asn" => (pages::render_asn(), "text/html; charset=utf-8"),
            "/streaming" => (pages::render_streaming(), "text/html; charset=utf-8"),
            "/router" => (pages::render_router(), "text/html; charset=utf-8"),
            "/graphs" => (pages::render_graphs(), "text/html; charset=utf-8"),
            "/sampling" => (pages::render_sampling(), "text/html; charset=utf-8"),
            "/cache" => (pages::render_cache(), "text/html; charset=utf-8"),
            "/flows" => (pages::render_flows(), "text/html; charset=utf-8"),
            "/login" => (pages::login::render_standalone(), "text/html; charset=utf-8"),
            _ => {
                let body = b"404 Not Found";
                let _ = stream.write_all(&http_response(404, "Not Found", "text/plain", body));
                let _ = stream.flush();
                return;
            }
        };
        http_response(200, "OK", ct, html.as_bytes())
    };

    let _ = stream.write_all(&response);
    let _ = stream.flush();
}

fn main() {
    let port = 8080;
    let addr = format!("0.0.0.0:{}", port);
    let listener = TcpListener::bind(&addr).expect("Failed to bind port");

    println!("╔══════════════════════════════════════════╗");
    println!("║          INFORFLOW v1.0.0                ║");
    println!("║   Network Flow Analysis System           ║");
    println!("║   Source: 170.245.127.191                ║");
    println!("╠══════════════════════════════════════════╣");
    println!("║  Dashboard:  http://localhost:{:<5}     ║", port);
    println!("║  Gráficos:   http://localhost:{}/graphs║", port);
    println!("║  Roteador:   http://localhost:{}/router║", port);
    println!("║  CDNs:       http://localhost:{}/cdns ║", port);
    println!("║  Streaming:  http://localhost:{}/streaming", port);
    println!("║  Peers:      http://localhost:{}/peers ║", port);
    println!("║  ASN:        http://localhost:{}/asn  ║", port);
    println!("╚══════════════════════════════════════════╝");

    for stream in listener.incoming() {
        if let Ok(stream) = stream {
            thread::spawn(move || handle_client(stream));
        }
    }
}
