use actix_ws::Message;
use bytestring::ByteString;
use deadpool_redis::{Connection, Pool};
use redis::aio::ConnectionLike;
use redis::RedisError;
use serde::{Deserialize, Serialize};
use std::sync::{Arc, RwLock};
use tokio::sync::mpsc::UnboundedSender;
use crate::errors::AppError;

type Client = UnboundedSender<Message>;


pub struct GridHolder {
    pub clients: Arc<RwLock<Vec<Client>>>,
    pub pool: Arc<Pool>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct DrawReq {
    pub x: usize,
    pub y: usize,
    pub color: u8,
}

impl GridHolder {
    /// Fetches the grid from Redis.
    pub async fn get_grid(&self) -> Result<Vec<u8>, AppError> {
        let mut conn = self.get_connection().await?;
        get_bitfield(&mut conn, "grid").await
            .map_err(AppError::from)
    }

    /// Updates a specific cell in the grid and notifies clients.
    pub async fn update_cell(&self, req: &DrawReq) -> Result<(), AppError> {
        let mut conn = self.get_connection().await?;
        let index = calculate_index(req.x, req.y);
        set_bit(&mut conn, "grid", index, req.color).await?;
        self.notify_clients(req);
        Ok(())
    }

    /// Acquires a connection from the pool.
    async fn get_connection(&self) -> Result<Connection, AppError> {
        self.pool.get().await
            .map_err(AppError::from)
    }

    /// Notifies all connected clients about a grid update.
    fn notify_clients(&self, req: &DrawReq) {
        let clients = self.clients.read().unwrap();
        if clients.is_empty() {
            return;
        }
        let message = match serde_json::to_string(req) {
            Ok(msg) => ByteString::from(msg),
            Err(e) => {
                eprintln!("Failed to serialize DrawReq: {}", e);
                return;
            }
        };

        for client in clients.iter() {
            if let Err(e) = client.send(Message::Text(message.clone())) {
                eprintln!("Failed to send message to client: {}", e);
            }
        }
    }
}

/// Creates a new GridHolder.
pub fn new(clients: Vec<Client>, pool: Pool) -> GridHolder {
    GridHolder {
        clients: Arc::new(RwLock::new(clients)),
        pool: Arc::new(pool),
    }
}

/// Sets a bit in the Redis bitfield.
async fn set_bit(conn: &mut impl ConnectionLike, key: &str, index: usize, value: u8) -> Result<(), RedisError> {
    redis::cmd("BITFIELD")
        .arg(key)
        .arg("SET")
        .arg("u4")
        .arg(index)
        .arg(value)
        .query_async(conn)
        .await
}

/// Retrieves the bitfield from Redis.
async fn get_bitfield(conn: &mut impl ConnectionLike, key: &str) -> Result<Vec<u8>, RedisError> {
    redis::cmd("GET")
        .arg(key)
        .query_async(conn)
        .await
}

/// Calculates the index for the bitfield based on x, y coordinates.
fn calculate_index(x: usize, y: usize) -> usize {
    (x + y * 500) * 4
}