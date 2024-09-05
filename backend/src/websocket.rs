use actix_web::{web, HttpRequest, Responder};
use actix_ws::Message;
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::unbounded_channel;

#[derive(Debug, Deserialize, Serialize)]
struct DrawReq {
    x: usize,
    y: usize,
    color: u8,
}

pub async fn ws(req: HttpRequest, body: web::Payload, state: web::Data<crate::holder::GridHolder>) -> actix_web::Result<impl Responder> {
    let (response, mut session, _msg_stream) = actix_ws::handle(&req, body)?;
    let (tx, mut rx) = unbounded_channel();

    state.clients.write().unwrap().push(tx);

    actix_web::rt::spawn(async move {
        while let Some(msg) = rx.recv().await {
            if let Message::Text(text) = msg {
                let _ = session.text(text).await;
            } else if let Message::Binary(bin) = msg {
                let _ = session.binary(bin).await;
            }
        }
    });

    Ok(response)
}