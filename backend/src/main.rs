mod websocket;
mod holder;
mod errors;

use std::env;

use crate::holder::{GridHolder, new, DrawReq};
use actix_web::middleware::Logger;
use actix_web::{web, App, HttpResponse, HttpServer, Responder};
use deadpool_redis::{Config, Pool, Runtime};
use futures_util::TryFutureExt;

#[actix_web::main]
async fn main() -> std::io::Result<()> {

    let redis_address = env::var("REDIS_ADDRESS").unwrap_or_else(|_| "redis://localhost:6379".to_string());
    let app_state = new(vec![], new_pool(&redis_address));
    let state = web::Data::new(app_state);

    env_logger::init_from_env(env_logger::Env::new().default_filter_or("debug"));
    let address = env::var("BIND_ADDRESS").unwrap_or_else(|_| "0.0.0.0:8080".to_string());

    HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(state.clone())
            .route("/api/ws", web::get().to(websocket::ws))
            .route("/api/grid", web::get().to(get_grid))
            .route("/api/draw", web::post().to(modify_cell))
    })
        .bind(address)?
        .run()
        .await
}

async fn get_grid(state: web::Data<GridHolder>) -> impl Responder {
    state.get_grid().map_ok_or_else(
        |e| HttpResponse::BadRequest().body(e.to_string()),
        |grid| HttpResponse::Ok().body(grid),
    ).await
}

async fn modify_cell(req: web::Json<DrawReq>, state: web::Data<GridHolder>) -> impl Responder {
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