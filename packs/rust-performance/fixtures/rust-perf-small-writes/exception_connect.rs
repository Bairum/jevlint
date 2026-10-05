use std::io::{Read, Write};

/// Connects to a device and performs its command-dependent calibration sequence.
/// The peer replies to each delivered byte; that reply is the next command byte.
/// Even long sessions require every command to arrive before waiting for its reply.
pub fn calibrate_device(
    address: std::net::SocketAddr,
    initial_command: u8,
    steps: usize,
) -> std::io::Result<u8> {
    let mut output = std::net::TcpStream::connect(address)?;
    output.set_nodelay(true)?;
    let mut command = initial_command;
    for _ in 0..steps {
        output.write_all(&[command])?;
        let mut reply = [0];
        output.read_exact(&mut reply)?;
        command = reply[0];
    }
    Ok(command)
}
