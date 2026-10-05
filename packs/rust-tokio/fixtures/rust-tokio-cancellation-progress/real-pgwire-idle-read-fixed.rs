use std::io;
use std::time::Duration;
use tokio::io::{AsyncReadExt, AsyncWriteExt, DuplexStream};

#[derive(Debug)]
pub struct BackendMessage {
    pub tag: u8,
    pub payload: Vec<u8>,
}

#[derive(Clone, Copy)]
pub struct ReaderConfig {
    pub idle_interval: Duration,
    pub maximum_payload: usize,
}

impl ReaderConfig {
    pub fn new(idle_interval: Duration, maximum_payload: usize) -> io::Result<Self> {
        if idle_interval.is_zero() || maximum_payload == 0 {
            return Err(io::ErrorKind::InvalidInput.into());
        }
        Ok(Self {
            idle_interval,
            maximum_payload,
        })
    }
}

#[derive(Default)]
pub struct Feedback {
    pub idle_ticks: u64,
    pub applied_position: u64,
}

impl Feedback {
    pub fn record_idle(&mut self) {
        self.idle_ticks = self.idle_ticks.saturating_add(1);
    }

    pub fn apply(&mut self, position: u64) {
        self.applied_position = self.applied_position.max(position);
    }

    pub fn encode(&self) -> [u8; 8] {
        self.applied_position.to_be_bytes()
    }
}

fn payload_length(header: &[u8; 5], maximum: usize) -> io::Result<usize> {
    let length = u32::from_be_bytes([header[1], header[2], header[3], header[4]]);
    let payload = length.checked_sub(4).ok_or(io::ErrorKind::InvalidData)? as usize;
    if payload > maximum {
        return Err(io::ErrorKind::InvalidData.into());
    }
    Ok(payload)
}

/// Returns the next complete backend message on this persistent replication stream.
/// Idle wakeups update feedback; every wire byte must belong to its original message.
pub async fn next_message(
    stream: &mut DuplexStream,
    config: ReaderConfig,
    feedback: &mut Feedback,
) -> io::Result<BackendMessage> {
    let mut header = [0_u8; 5];
    let mut header_filled = 0;
    while header_filled < header.len() {
        let operation = stream.read(&mut header[header_filled..]);
        match tokio::time::timeout(config.idle_interval, operation).await {
            Ok(result) => {
                let count = result?;
                if count == 0 {
                    return Err(io::ErrorKind::UnexpectedEof.into());
                }
                header_filled += count;
            }
            Err(_) => feedback.record_idle(),
        }
    }
    let length = payload_length(&header, config.maximum_payload)?;
    let mut payload = vec![0_u8; length];
    let mut payload_filled = 0;
    while payload_filled < payload.len() {
        let operation = stream.read(&mut payload[payload_filled..]);
        match tokio::time::timeout(config.idle_interval, operation).await {
            Ok(result) => {
                let count = result?;
                if count == 0 {
                    return Err(io::ErrorKind::UnexpectedEof.into());
                }
                payload_filled += count;
            }
            Err(_) => feedback.record_idle(),
        }
    }
    Ok(BackendMessage {
        tag: header[0],
        payload,
    })
}

pub async fn read_startup(stream: &mut DuplexStream) -> io::Result<[u8; 8]> {
    let mut greeting = [0_u8; 8];
    stream.read_exact(&mut greeting).await?;
    Ok(greeting)
}

pub async fn write_feedback(stream: &mut DuplexStream, feedback: &Feedback) -> io::Result<()> {
    stream.write_all(&feedback.encode()).await
}

pub fn replication_position(message: &BackendMessage) -> Option<u64> {
    if message.tag != b'w' || message.payload.len() < 8 {
        return None;
    }
    let mut bytes = [0_u8; 8];
    bytes.copy_from_slice(&message.payload[..8]);
    Some(u64::from_be_bytes(bytes))
}

pub fn startup_request(position: u64) -> Vec<u8> {
    let mut request = Vec::with_capacity(13);
    request.push(b'S');
    request.extend_from_slice(&12_u32.to_be_bytes());
    request.extend_from_slice(&position.to_be_bytes());
    request
}
