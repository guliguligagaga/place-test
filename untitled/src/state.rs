use std::sync::{Arc, RwLock};
use actix_ws::Message;
use bytestring::ByteString;
use deadpool_redis::{Config, Pool, Runtime};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::UnboundedSender;
use tokio::sync::Mutex;
use crate::grid::Grid;

type Client = UnboundedSender<Message>;

pub struct AppState {
    // pub grid: Arc<RwLock<Grid>>,
    pub clients: Arc<RwLock<Vec<Client>>>,
    pub pool: Pool,
}

#[derive(Debug, Deserialize, Serialize)]
struct DrawReq {
    x: usize,
    y: usize,
    color: u8,
}

impl AppState {
    pub async fn new(clients: Vec<Client>) -> Self {
        AppState {
            // grid,
            clients: Arc::new(RwLock::new(clients)),
            pool: new("redis://localhost:6379").await,
        }
    }

    pub fn notify_clients(&self, x: usize, y: usize, color: u8) {
        let clients = self.clients.read().unwrap();
        if clients.iter().len() == 0 {
            return;
        }
        let req = &(DrawReq { x, y, color });
        let message = ByteString::from(serde_json::to_string(req).unwrap());

        for client in clients.iter() {
            let _ = client.send(Message::Text(message.clone()));
        }
    }
}


pub async fn new(redis_url: &str) -> Pool {
    let cfg = Config::from_url(redis_url);
    cfg.create_pool(Option::from(Runtime::Tokio1)).expect("Failed to create Redis pool")
}

pub async fn set_bit(pool: &Pool, key: &str, index: usize, value: u8) -> redis::RedisResult<()> {
    let mut conn = pool.get().await.expect("Failed to get Redis connection");
    redis::cmd("BITFIELD")
        .arg(key)
        .arg("SET")
        .arg("u4")
        .arg(index)
        .arg(value)
        .query_async(&mut conn)
        .await?;
    Ok(())
}

pub async fn get_bit(pool: &Pool, key: &str, index: usize) -> u8 {
    let mut conn = pool.get().await.expect("Failed to get Redis connection");
    redis::cmd("BITFIELD")
        .arg(key)
        .arg("GET")
        .arg("u4")
        .arg(index)
        .query_async(&mut conn)
        .await.unwrap()
}

pub async fn get_bitfield(pool: &Pool, key: &str) -> Vec<u8> {
    let mut conn = pool.get().await.expect("Failed to get Redis connection");
    redis::cmd("GET")
        .arg(key)
        .query_async(&mut conn)
        .await.unwrap()
}