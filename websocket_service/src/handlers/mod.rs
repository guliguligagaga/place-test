pub mod websocket;
pub mod kafka;
pub mod grid;

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::RwLock;
use warp::ws::Message;

use crate::error::ServiceError;

type Clients = Arc<RwLock<std::collections::HashMap<String, tokio::sync::mpsc::Sender<Message>>>>;

#[async_trait]
pub trait WebSocketHandler: Send + Sync {
    async fn handle_message(&self, msg: Message, client_id: &str, clients: &Clients) -> Result<(), ServiceError>;
    async fn on_disconnect(&self, client_id: &str, clients: &Clients);
}

#[async_trait]
pub trait KafkaHandler: Send + Sync {
    async fn handle_message(&self, msg: &rdkafka::message::BorrowedMessage<'_>, clients: &Clients) -> Result<(), ServiceError>;
}