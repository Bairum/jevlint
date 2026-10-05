use std::io::{self, Cursor, Read};
use std::net::{SocketAddr, UdpSocket};

pub fn receive_envelope(socket: &UdpSocket) -> io::Result<(SocketAddr, Vec<u8>)> {
    let mut packet = [0_u8; 4096];
    let (length, peer) = socket.recv_from(&mut packet)?;
    let mut input = Cursor::new(&packet[..length]);
    let mut payload = Vec::new();
    input.read_to_end(&mut payload)?;
    Ok((peer, payload))
}

