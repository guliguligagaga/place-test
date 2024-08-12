// src/websocket.rs

use std::sync::{Arc, RwLock};
use actix_web::{web, HttpRequest, Responder};
use actix_ws::Message;
use futures_util::stream::StreamExt;
use crate::grid::Grid;

pub async fn ws(req: HttpRequest, body: web::Payload, grid: web::Data<Arc<RwLock<Grid>>>) -> actix_web::Result<impl Responder> {
    let (response, mut session, mut msg_stream) = actix_ws::handle(&req, body)?;

    actix_web::rt::spawn(async move {
        while let Some(Ok(msg)) = msg_stream.next().await {
            match msg {
                Message::Text(text) => {
                    let parts: Vec<&str> = text.split(',').collect();
                    if parts.len() == 2 {
                        if let (Ok(x), Ok(y)) = (parts[0].parse::<usize>(), parts[1].parse::<usize>()) {
                            let grid = grid.try_read().unwrap();
                            if let Some(cell) = grid.get_cell(x, y) {
                                let response = serde_json::to_string(&cell).unwrap();
                                session.binary(response).await.unwrap()
                            }
                        }
                    }
                }
                Message::Ping(bytes) => {
                    if session.pong(&bytes).await.is_err() {
                        return;
                    }
                }
                _ => break,
            }
        }

        let _ = session.close(None).await;
    });

    Ok(response)
}
