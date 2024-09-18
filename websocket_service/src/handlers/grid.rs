use rdkafka::message::Message as KafkaMessage;
use warp::ws::Message;
use crate::Clients;
use crate::error::ServiceError;
use crate::handlers::{KafkaHandler, WebSocketHandler};

pub struct GridWebSocketHandler;

impl GridWebSocketHandler {
    pub fn new() -> Self {
        GridWebSocketHandler
    }
}

#[async_trait::async_trait]
impl WebSocketHandler for GridWebSocketHandler {
    async fn handle_message(&self, msg: Message, client_id: &str, clients: &Clients) -> Result<(), ServiceError> {
        //TODO 
        Ok(())
    }

    async fn on_disconnect(&self, client_id: &str, clients: &Clients) {
        //TODO
    }
}

pub struct GridKafkaMessageHandler;

impl GridKafkaMessageHandler {
    pub fn new() -> Self {
        GridKafkaMessageHandler
    }
}

#[async_trait::async_trait]
impl KafkaHandler for GridKafkaMessageHandler {
    async fn handle_message(&self, payload: &rdkafka::message::BorrowedMessage<'_>, clients: &Clients) -> Result<(), ServiceError> {
        let update = String::from_utf8_lossy(payload.payload().unwrap());
        let clients_read = clients.read().await;
        for (_, client_sender) in clients_read.iter() {
            client_sender.send(Message::text(update.clone())).await?;
        }
        Ok(())
    }
}