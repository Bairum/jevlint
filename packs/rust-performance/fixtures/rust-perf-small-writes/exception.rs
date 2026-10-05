/// Drives an interactive device protocol on an established connection.
/// The peer waits for one command byte before sending one response byte.
/// Each response determines the next command, requiring delivery before the read.
/// On I/O error the caller discards the connection rather than restarting an exchange.
pub fn exchange_commands(
    stream: &mut std::net::TcpStream,
    initial_command: u8,
    steps: usize,
) -> std::io::Result<u8> {
    std::net::TcpStream::set_nodelay(stream, true)?;
    let mut command = initial_command;
    for _ in 0..steps {
        <std::net::TcpStream as std::io::Write>::write_all(stream, &[command])?;
        let mut response = [0_u8; 1];
        <std::net::TcpStream as std::io::Read>::read_exact(stream, &mut response)?;
        command = response[0];
    }
    Ok(command)
}
