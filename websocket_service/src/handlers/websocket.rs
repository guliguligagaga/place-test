use std::sync::Arc;
use super::{Clients, WebSocketHandler};
use futures::StreamExt;
use tokio::sync::{broadcast, mpsc};
use tracing::{error, warn};
use warp::ws::{Message, WebSocket};
use tokio_stream::wrappers::ReceiverStream;
const MAX_CONNECTIONS: usize = 10000;
const CHANNEL_BUFFER_SIZE: usize = 100;

pub async fn handle_client(
    ws: WebSocket,
    clients: Clients,
    handler: Arc<dyn WebSocketHandler>,
    mut shutdown_signal: broadcast::Receiver<()>,
) {
    let (client_ws_sender, mut client_ws_rcv) = ws.split();
    let (client_sender, client_rcv) = mpsc::channel(CHANNEL_BUFFER_SIZE);

    let client_id = uuid::Uuid::new_v4().to_string();
    let client_id_clone = client_id.clone();
    let clients_clone = clients.clone();
    {
        let mut clients_write = clients.write().await;
        if clients_write.len() >= MAX_CONNECTIONS {
            error!("Connection limit reached");
            return;
        }
        clients_write.insert(client_id.clone(), client_sender);
    }

    let mut send_task = tokio::task::spawn(ReceiverStream::new(client_rcv)
        .map(Ok)
        .forward(client_ws_sender));
    let handler_clone = Arc::clone(&handler);
    let mut recv_task = tokio::task::spawn(async move {
        while let Some(result) = tokio::select! {
            result = client_ws_rcv.next() => result,
            _ = shutdown_signal.recv() => None,
        } {
            match result {
                Ok(msg) => {
                    if let Err(e) = handler_clone.handle_message(msg, &client_id_clone, &clients_clone).await {
                        error!("Error handling message: {}", e);
                        break;
                    }
                }
                Err(e) => {
                    warn!("WebSocket error: {}", e);
                    break;
                }
            }
        }
    });

    tokio::select! {
        _ = (&mut send_task) => recv_task.abort(),
        _ = (&mut recv_task) => send_task.abort(),
    }

    handler.on_disconnect(&client_id, &clients).await;
    clients.write().await.remove(&client_id);
}
