use deadpool_redis::{Connection, Pool};
use redis::aio::ConnectionLike;
use redis::RedisError;
use serde::{Deserialize, Serialize};
use std::sync::{Arc, RwLock};
use std::collections::{HashMap, HashSet};
use tokio::sync::mpsc::UnboundedSender;
use crate::errors::AppError;
use std::time::{Duration, Instant};

type Client = UnboundedSender<String>;

const QUADRANT_SIZE: usize = 50;
const GRID_SIZE: usize = 100; // Assuming a 2000x2000 grid
const INACTIVITY_TIMEOUT: Duration = Duration::from_secs(300); // 5 minutes

pub struct GridHolder {
    clients: RwLock<HashMap<usize, (Client, Instant)>>,
    quadrant_subscriptions: RwLock<HashMap<usize, HashSet<usize>>>,
    pool: Pool,
    next_client_id: RwLock<usize>,
    quadrants: RwLock<Vec<Quadrant>>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct DrawReq {
    pub x: usize,
    pub y: usize,
    pub color: u8,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct Quadrant {
    pub id: usize,
    pub x: usize,
    pub y: usize,
}

#[derive(Debug, Serialize)]
struct ConfigurationMessage {
    r#type: String,
    quadrants: Vec<Quadrant>,
}

#[derive(Debug, Serialize)]
struct UpdateMessage {
    r#type: String,
    x: usize,
    y: usize,
    color: u8,
}

impl GridHolder {
    pub async fn get_grid(self: &Arc<Self>) -> Result<Vec<u8>, AppError> {
        let mut conn = self.get_connection().await?;
        let result = get_bitfield(&mut conn, "grid").await;
        match &result {
            Ok(grid) => println!("Retrieved grid from Redis: {} bytes", grid.len()),
            Err(e) => eprintln!("Error retrieving grid from Redis: {:?}", e),
        }
        result.map_err(AppError::from)
    }

    pub async fn update_cell(self: &Arc<Self>, req: &DrawReq) -> Result<(), AppError> {
        let mut conn = self.get_connection().await?;
        let index = calculate_index(req.x, req.y);
        set_bit(&mut conn, "grid", index, req.color).await?;
        self.notify_quadrant(req);
        Ok(())
    }

    async fn get_connection(&self) -> Result<Connection, AppError> {
        self.pool.get().await
            .map_err(AppError::from)
    }

    fn notify_quadrant(&self, req: &DrawReq) {
        let quadrant_id = calculate_quadrant_id(req.x, req.y);
        let subscriptions = self.quadrant_subscriptions.read().unwrap();
        let clients = self.clients.read().unwrap();

        if let Some(subscribed_clients) = subscriptions.get(&quadrant_id) {
            let update_message = UpdateMessage {
                r#type: "update".to_string(),
                x: req.x,
                y: req.y,
                color: req.color,
            };
            let message = serde_json::to_string(&update_message).unwrap();
            for &client_id in subscribed_clients {
                if let Some((client, _)) = clients.get(&client_id) {
                    let _ = client.send(message.clone());
                }
            }
        }
    }

    pub async fn initialize_grid(self: &Arc<Self>) -> Result<(), AppError> {
        let mut conn = self.get_connection().await?;
        let exists: bool = redis::cmd("EXISTS")
            .arg("grid")
            .query_async(&mut conn)
            .await?;

        if !exists {
            println!("Grid does not exist in Redis. Initializing empty grid.");
            let empty_grid = vec![0u8; GRID_SIZE * GRID_SIZE / 2]; // 4 bits per cell
            redis::cmd("SET")
                .arg("grid")
                .arg(empty_grid)
                .query_async(&mut conn)
                .await?;
        } else {
            println!("Grid already exists in Redis.");
        }
        Ok(())
    }

    pub fn remove_client(&self, client_id: usize) {
        self.clients.write().unwrap().remove(&client_id);
        let mut subscriptions = self.quadrant_subscriptions.write().unwrap();
        for subscribed_clients in subscriptions.values_mut() {
            subscribed_clients.remove(&client_id);
        }
    }

    pub fn subscribe_to_quadrant(&self, client_id: usize, quadrant_id: usize) {
        self.quadrant_subscriptions
            .write()
            .unwrap()
            .entry(quadrant_id)
            .or_insert_with(HashSet::new)
            .insert(client_id);
    }

    pub fn unsubscribe_from_quadrant(&self, client_id: usize, quadrant_id: usize) {
        if let Some(subscribed_clients) = self.quadrant_subscriptions.write().unwrap().get_mut(&quadrant_id) {
            subscribed_clients.remove(&client_id);
        }
    }

    pub fn update_client_activity(&self, client_id: usize) {
        if let Some((_, last_activity)) = self.clients.write().unwrap().get_mut(&client_id) {
            *last_activity = Instant::now();
        }
    }

    pub fn clean_inactive_clients(&self) {
        let now = Instant::now();
        let mut clients_to_remove = Vec::new();

        {
            let clients = self.clients.read().unwrap();
            for (&client_id, (_, last_activity)) in clients.iter() {
                if now.duration_since(*last_activity) > INACTIVITY_TIMEOUT {
                    clients_to_remove.push(client_id);
                }
            }
        }

        for client_id in clients_to_remove {
            self.remove_client(client_id);
        }
    }

    pub fn add_client(&self, client: Client) -> usize {
        let mut next_id = self.next_client_id.write().unwrap();
        let id = *next_id;
        *next_id += 1;
        self.clients.write().unwrap().insert(id, (client.clone(), Instant::now()));

        // Send configuration message to the new client
        let config_message = ConfigurationMessage {
            r#type: "configuration".to_string(),
            quadrants: self.quadrants.read().unwrap().clone(),
        };
        let message = serde_json::to_string(&config_message).unwrap();
        let _ = client.send(message);

        id
    }

    pub fn initialize_quadrants(&self) {
        let mut quadrants = Vec::new();
        for y in (0..GRID_SIZE).step_by(QUADRANT_SIZE) {
            for x in (0..GRID_SIZE).step_by(QUADRANT_SIZE) {
                quadrants.push(Quadrant {
                    id: calculate_quadrant_id(x, y),
                    x,
                    y,
                });
            }
        }
        *self.quadrants.write().unwrap() = quadrants;
    }
}

pub fn new(pool: Pool) -> GridHolder {
    let holder = GridHolder {
        clients: RwLock::new(HashMap::new()),
        quadrant_subscriptions: RwLock::new(HashMap::new()),
        pool,
        next_client_id: RwLock::new(0),
        quadrants: RwLock::new(Vec::new()),
    };
    holder.initialize_quadrants();
    holder
}

async fn set_bit(conn: &mut impl ConnectionLike, key: &str, index: usize, value: u8) -> Result<(), RedisError> {
    let bit_offset = index * 4;  // Each cell is 4 bits

    println!("Setting bit: index={}, bit_offset={}, value={}",
             index, bit_offset, value);
    redis::cmd("BITFIELD")
        .arg(key)
        .arg("SET")
        .arg("u4")
        .arg(index * 4)
        .arg(value)
        .query_async(conn)
        .await
}

async fn get_bitfield(conn: &mut impl ConnectionLike, key: &str) -> Result<Vec<u8>, RedisError> {
    redis::cmd("GET")
        .arg(key)
        .query_async(conn)
        .await
}

fn calculate_index(x: usize, y: usize) -> usize {
    x + y * GRID_SIZE
}

fn calculate_quadrant_id(x: usize, y: usize) -> usize {
    (x / QUADRANT_SIZE) + (y / QUADRANT_SIZE) * (GRID_SIZE / QUADRANT_SIZE)
}