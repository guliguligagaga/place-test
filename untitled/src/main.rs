mod websocket;
mod grid;

use actix_web::{web, App, HttpServer, HttpResponse, Responder};
use serde::{Deserialize};
use std::sync::{Arc, RwLock};
use actix_web::middleware::Logger;
use grid::Grid;

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
