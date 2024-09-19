use std::sync::Arc;
use super::{Clients, KafkaHandler};
use crate::error::ServiceError;
use rdkafka::consumer::{StreamConsumer};
use tokio::sync::broadcast;
use tracing::error;

pub async fn handle_kafka_messages(
    consumer: StreamConsumer,
    clients: Clients,
    handler: Arc<dyn KafkaHandler>,
    mut shutdown_signal: broadcast::Receiver<()>,
) -> Result<(), ServiceError> {
    loop {
        tokio::select! {
            message = consumer.recv() => {
                match message {
                    Ok(msg) => {
                        if let Err(e) = handler.handle_message(&msg, &clients).await {
                            error!("Error handling Kafka message: {}", e);
                        }
                    }
                    Err(e) => {
                        error!("Error receiving Kafka message: {}", e);
                    }
                }
            }
            _ = shutdown_signal.recv() => {
                break;
            }
        }
    }
    Ok(())
}