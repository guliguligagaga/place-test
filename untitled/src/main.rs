mod websocket;
mod grid;

use actix_web::{web, App, HttpServer, HttpResponse, Responder};
use serde::Deserialize;
use std::sync::{Arc, RwLock};
use actix_web::middleware::Logger;
use grid::Grid;

/// Retrieves the current state of the grid as a bitfield, wrapped in an HTTP response.
///
/// This asynchronous function reads the grid from a shared, thread-safe container and converts it
/// into a bitfield representation. It then wraps this bitfield into an HTTP response with a status
/// code of 200 (OK).
///
/// # Arguments
///
/// * `grid` - A web data wrapper containing an `Arc` wrapped `RwLock` of the `Grid` struct. This
///   ensures that the grid can be shared and read across multiple threads safely.
///
/// # Returns
///
/// An implementation of the `Responder` trait containing the bitfield representation of the grid
/// as the HTTP response body.
async fn get_grid(grid: web::Data<Arc<RwLock<Grid>>>) -> impl Responder {
    let grid = grid.read().unwrap();
    HttpResponse::Ok().body(grid.to_bitfield())
}

#[derive(Debug, Deserialize)]
struct DrawReq {
    x: usize,
    y: usize,
    color: u8,
}

async fn modify_cell(req: web::Json<DrawReq>, grid: web::Data<Arc<RwLock<Grid>>>) -> impl Responder {
    let mut grid = grid.write().unwrap();
    if grid.modify_cell(req.x, req.y, req.color).is_some() {
        HttpResponse::Ok().body("Cell updated")
    } else {
        HttpResponse::BadRequest().body("Invalid cell coordinates")
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let grid = Arc::new(RwLock::new(Grid::new()));

    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));
    HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(web::Data::new(grid.clone()))
            .route("/ws/", web::get().to(websocket::ws))
            .route("/api/grid", web::get().to(get_grid))
            .route("/api/draw", web::post().to(modify_cell))
    })
        .bind("127.0.0.1:8080")?
        .run()
        .await
}
