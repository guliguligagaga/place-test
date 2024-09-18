mod error;
mod handlers;
mod config;

use crate::handlers::{websocket::handle_client, kafka::handle_kafka_messages, KafkaHandler, WebSocketHandler};
use crate::config::get_kafka_config;
use crate::handlers::{grid::GridKafkaMessageHandler, grid::GridWebSocketHandler};

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{mpsc, RwLock, broadcast};
use warp::ws::{Message, WebSocket};
use warp::Filter;
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{StreamConsumer, Consumer};
use rdkafka::message::Message as KafkaMessage;
use futures::StreamExt;
use thiserror::Error;
use tracing::{info, error, warn, instrument};
use serde::{Deserialize, Serialize};
use async_trait::async_trait;

type Clients = Arc<RwLock<HashMap<String, mpsc::Sender<Message>>>>;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {

    let clients: Clients = Arc::new(RwLock::new(HashMap::new()));
    let clients_clone = clients.clone();
    let clients_filter = warp::any().map(move || clients.clone());

    let ws_handler = Arc::new(GridWebSocketHandler::new());

    let (shutdown_sender, _) = broadcast::channel::<()>(1);
    let shutdown_sender_clone_route = shutdown_sender.clone();
    let shutdown_sender_clone_bind = shutdown_sender.clone();

    let ws_route = warp::path("ws")
        .and(warp::ws())
        .and(clients_filter)
        .and(warp::any().map(move || ws_handler.clone()))
        .and(warp::any().map(move || shutdown_sender_clone_route.subscribe()))
        .map(|ws: warp::ws::Ws, clients, handler: Arc<GridWebSocketHandler>, shutdown| {
            ws.on_upgrade(move |socket| handle_client(socket, clients, handler, shutdown))
        });
    //kafka
    let kafka_config = get_kafka_config()?;
    let kafka_consumer: StreamConsumer = kafka_config.create()?;

    kafka_consumer.subscribe(&["grid_updates"])?;

    let kafka_handler = Arc::new(GridKafkaMessageHandler::new());
    let kafka_handler_clone = kafka_handler.clone();
    
    let kafka_shutdown = shutdown_sender.clone().subscribe();
    tokio::spawn(async move {
        if let Err(e) = handle_kafka_messages(kafka_consumer, clients_clone, kafka_handler_clone, kafka_shutdown).await {
            error!("Kafka message handling error: {}", e);
        }
    });
    
    // server
    let server = warp::serve(ws_route);
    let (_, server_future) = server.bind_with_graceful_shutdown(([127, 0, 0, 1], 8081), async move {
        shutdown_sender_clone_bind.subscribe().recv().await.ok();
    });

    tokio::select! {
        _ = server_future => {},
        _ = tokio::signal::ctrl_c() => {
            info!("Shutdown signal received");
            let _ = shutdown_sender.send(());
        }
    }

    Ok(())
}