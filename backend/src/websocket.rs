use actix_web::{web, HttpRequest, HttpResponse};
use actix_ws::Message;
use futures_util::StreamExt;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use std::sync::Arc;
use crate::holder::{GridHolder, DrawReq};

#[derive(Debug, Deserialize, Serialize)]
#[serde(tag = "type", content = "payload")]
enum WsMessage {
    Subscribe { quadrant_id: usize },
    Unsubscribe { quadrant_id: usize },
    Draw(DrawReq),
    Activity,
}

pub async fn ws(req: HttpRequest, body: web::Payload, state: web::Data<Arc<GridHolder>>) -> actix_web::Result<HttpResponse> {
    let (response, mut session, mut msg_stream) = actix_ws::handle(&req, body)?;
    let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel();

    println!("New client connected");
    let client_id = state.add_client(tx.clone());

    actix_web::rt::spawn(async move {
        let mut subscribed_quadrants = HashSet::new();

        while let Some(Ok(msg)) = msg_stream.next().await {
            match msg {
                Message::Text(text) => {
                    match serde_json::from_str::<WsMessage>(&text) {
                        Ok(ws_msg) => {
                            match ws_msg {
                                WsMessage::Subscribe { quadrant_id } => {
                                    //state.subscribe_to_quadrant(client_id, quadrant_id);
                                    subscribed_quadrants.insert(quadrant_id);
                                },
                                WsMessage::Unsubscribe { quadrant_id } => {
                                    //state.unsubscribe_from_quadrant(client_id, quadrant_id);
                                    subscribed_quadrants.remove(&quadrant_id);
                                },
                                WsMessage::Draw(draw_req) => {
                                    if let Err(e) = state.update_cell(&draw_req).await {
                                        eprintln!("Failed to update cell: {:?}", e);
                                    }
                                },
                                WsMessage::Activity => {
                                    state.update_client_activity(client_id);
                                },
                            }
                        },
                        Err(e) => {
                            eprintln!("Failed to parse WebSocket message: {:?}", e);
                        }
                    }
                },
                Message::Close(_) => break,
                _ => {}
            }
        }

        // // Clean up when the connection is closed
        // for quadrant_id in subscribed_quadrants {
        //     state.unsubscribe_from_quadrant(client_id, quadrant_id);
        // }
        state.remove_client(client_id);
    });

    actix_web::rt::spawn(async move {
        while let Some(msg) = rx.recv().await {
            if let Err(e) = session.text(msg).await {
                eprintln!("Failed to send message to client: {:?}", e);
                break;
            }
        }
    });

    Ok(response)
}