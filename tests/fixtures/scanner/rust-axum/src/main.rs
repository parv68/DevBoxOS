use axum::{routing::get, Router};
use std::net::TcpListener;

#[tokio::main]
async fn main() {
    let app = Router::new().route("/", get(|| async { "Hello" }));
    let listener = TcpListener::bind("127.0.0.1:3000").unwrap();
    axum::serve(listener, app).await.unwrap();
}
