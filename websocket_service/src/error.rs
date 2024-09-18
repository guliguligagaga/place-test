use thiserror::Error;

#[derive(Error, Debug)]
pub enum ServiceError {
    #[error("WebSocket error: {0}")]
    WebSocketError(#[from] warp::Error),
    #[error("Kafka error: {0}")]
    KafkaError(#[from] rdkafka::error::KafkaError),
    #[error("Channel send error: {0}")]
    ChannelSendError(#[from] tokio::sync::mpsc::error::SendError<warp::ws::Message>),
    #[error("Connection limit reached")]
    ConnectionLimitReached,
    #[error("Invalid client configuration: {0}")]
    InvalidClientConfig(String),
    #[error("Unsupported handler type: {0}")]
    UnsupportedHandlerType(String),
}