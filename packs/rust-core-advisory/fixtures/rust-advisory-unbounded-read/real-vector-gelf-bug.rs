use std::io::{self, Read, Write};
use std::net::{IpAddr, SocketAddr};
use std::time::SystemTime;

pub struct IngestConfig {
    pub max_expanded_bytes: usize,
    pub allowed_peer: Option<IpAddr>,
}

pub struct Event {
    pub peer: SocketAddr,
    pub received_at: SystemTime,
    pub payload: Vec<u8>,
}

#[derive(Default)]
pub struct IngestStats {
    pub accepted: u64,
    pub rejected: u64,
    pub payload_bytes: u64,
}

impl IngestConfig {
    pub fn validate(&self) -> io::Result<()> {
        if self.max_expanded_bytes == 0 || self.max_expanded_bytes > 16 * 1024 * 1024 {
            return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid event size"));
        }
        Ok(())
    }

    pub fn permits(&self, peer: SocketAddr) -> bool {
        self.allowed_peer.map_or(true, |allowed| allowed == peer.ip())
    }
}

/// The decoder yields expanded bytes from a network-supplied GELF message.
pub fn read_expanded_message<R: Read>(
    mut decoder: R,
    peer: SocketAddr,
    config: &IngestConfig,
) -> io::Result<Event> {
    config.validate()?;
    if !config.permits(peer) {
        return Err(io::Error::new(io::ErrorKind::PermissionDenied, "peer not permitted"));
    }
    let mut payload = Vec::new();
    decoder.read_to_end(&mut payload)?;
    if payload.len() > config.max_expanded_bytes {
        return Err(io::Error::new(io::ErrorKind::InvalidData, "event exceeds size limit"));
    }
    if payload.is_empty() {
        return Err(io::Error::new(io::ErrorKind::InvalidData, "empty event"));
    }
    Ok(Event {
        peer,
        received_at: SystemTime::now(),
        payload,
    })
}

pub fn write_event<W: Write>(event: &Event, sink: &mut W) -> io::Result<()> {
    let length = u32::try_from(event.payload.len())
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "event too large"))?;
    sink.write_all(&length.to_be_bytes())?;
    sink.write_all(&event.payload)?;
    sink.flush()
}

pub fn ingest<R: Read, W: Write>(
    decoder: R,
    peer: SocketAddr,
    config: &IngestConfig,
    stats: &mut IngestStats,
    sink: &mut W,
) -> io::Result<()> {
    let event = match read_expanded_message(decoder, peer, config) {
        Ok(event) => event,
        Err(error) => {
            stats.rejected = stats.rejected.saturating_add(1);
            return Err(error);
        }
    };
    write_event(&event, sink)?;
    stats.accepted = stats.accepted.saturating_add(1);
    stats.payload_bytes = stats.payload_bytes.saturating_add(event.payload.len() as u64);
    Ok(())
}

