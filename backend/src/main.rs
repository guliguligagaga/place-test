mod websocket;
mod holder;
mod errors;

use std::env;
use std::sync::Arc;
use std::time::Duration;

use crate::holder::{GridHolder, new, DrawReq};
use actix_web::middleware::Logger;
use actix_web::{web, App, HttpResponse, HttpServer, Responder};
use deadpool_redis::{Config, Pool, Runtime};
use futures_util::TryFutureExt;
use tokio::sync::oneshot;
use tokio::signal;
use tokio::time::interval;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let redis_address = env::var("REDIS_ADDRESS").unwrap_or_else(|_| "redis://localhost:6379".to_string());
    let app_state = Arc::new(new(new_pool(&redis_address)));
    let state = web::Data::new(app_state.clone());

    app_state.initialize_grid().await.expect("Failed to initialize grid");

    env_logger::init_from_env(env_logger::Env::new().default_filter_or("debug"));
    let address = env::var("BIND_ADDRESS").unwrap_or_else(|_| "0.0.0.0:8080".to_string());

    let (shutdown_tx, shutdown_rx) = oneshot::channel();

    let server = HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(state.clone())
            .route("/ws", web::get().to(websocket::ws))
            .route("/grid", web::get().to(get_grid))
            .route("/draw", web::post().to(modify_cell))
    })
        .bind(address)?
        .shutdown_timeout(30) // Allow 30 seconds for graceful shutdown
        .run();

    let srv = server.handle();

    let cleanup_state = app_state.clone();
    tokio::spawn(async move {
        let mut interval = interval(Duration::from_secs(60)); // Run every minute
        loop {
            interval.tick().await;
            cleanup_state.clean_inactive_clients();
        }
    });

    tokio::spawn(async move {
        signal::ctrl_c().await.expect("Failed to listen for ctrl+c");
        println!("Received shutdown signal");
        srv.stop(true).await;
        shutdown_tx.send(()).expect("Failed to send shutdown signal");
    });

    println!("Server running. Press Ctrl-C to stop");
    server.await?;

    // Wait for the shutdown signal
    shutdown_rx.await.expect("Failed to receive shutdown signal");

    println!("Server shut down gracefully");
    Ok(())
}

async fn get_grid(state: web::Data<Arc<GridHolder>>) -> impl Responder {
    state.get_grid().map_ok_or_else(
        |e| HttpResponse::BadRequest().body(e.to_string()),
        |grid| HttpResponse::Ok().body(grid),
    ).await
}

async fn modify_cell(req: web::Json<DrawReq>, state: web::Data<Arc<GridHolder>>) -> impl Responder {
    state.update_cell(&req)
        .map_ok_or_else(
            |e| HttpResponse::BadRequest().body(e.to_string()),
            |_| HttpResponse::Ok().body("{\"status\": \"ok\"}"),
        ).await
}

fn new_pool(redis_url: &str) -> Pool {
    let cfg = Config::from_url(redis_url);
    cfg.create_pool(Option::from(Runtime::Tokio1)).expect("Failed to create Redis pool")
}