mod grid;
mod websocket;
mod state;

use actix_web::{web, App, HttpServer, HttpResponse, Responder, HttpRequest};
use serde::Deserialize;
use actix_web::middleware::Logger;
use crate::state::{set_bit, AppState};

async fn get_grid(state: web::Data<AppState>) -> impl Responder {
    let grid = state::get_bitfield(&state.pool, "grid").await;
    HttpResponse::Ok().body(grid)
}

#[derive(Debug, Deserialize)]
struct DrawReq {
    x: usize,
    y: usize,
    color: u8,
}

async fn modify_cell(req: web::Json<DrawReq>, state: web::Data<AppState>) -> impl Responder {
    match set_bit(&state.pool,"grid", (req.x + req.y * 500)*4, req.color).await {
        Ok(_) => {
            state.notify_clients(req.x, req.y, req.color);
            HttpResponse::Ok().body("Cell updated")
        }
        Err(e) => HttpResponse::BadRequest().body(e.to_string()),
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let app_state = AppState::new(vec![]).await;
    let state = web::Data::new(app_state);

    env_logger::init_from_env(env_logger::Env::new().default_filter_or("debug"));
    HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(state.clone())
            .route("/api/ws", web::get().to(websocket::ws))
            .route("/api/grid", web::get().to(get_grid))
            .route("/api/draw", web::post().to(modify_cell))
    })
        .bind("0.0.0.0:8080")?
        .run()
        .await
}