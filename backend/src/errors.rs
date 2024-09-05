use std::fmt;
use deadpool_redis::PoolError;
use redis::RedisError;
use serde::de::StdError;

#[derive(Debug)]
pub enum AppError {
    RedisError(RedisError),
    SerializationError(serde_json::Error),
    Other(String),
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::RedisError(e) => write!(f, "Redis error: {}", e),
            AppError::SerializationError(e) => write!(f, "Serialization error: {}", e),
            AppError::Other(msg) => write!(f, "Error: {}", msg),
        }
    }
}

impl StdError for AppError {
    fn source(&self) -> Option<&(dyn StdError + 'static)> {
        match self {
            AppError::RedisError(e) => Some(e),
            AppError::SerializationError(e) => Some(e),
            AppError::Other(_) => None,
        }
    }
}

impl From<RedisError> for AppError {
    fn from(e: RedisError) -> Self {
        AppError::RedisError(e)
    }
}

impl From<serde_json::Error> for AppError {
    fn from(e: serde_json::Error) -> Self {
        AppError::SerializationError(e)
    }
}

impl From<PoolError> for AppError {
    fn from(e: PoolError) -> Self {
        AppError::Other(format!("Deadpool error: {}", e))
    }
}