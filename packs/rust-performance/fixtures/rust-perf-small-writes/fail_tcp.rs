use std::io::Write;

/// Uploads 100,000 fixed-width records to the bulk-ingestion service.
/// The service receives the entire batch before issuing any reply.
/// Batch throughput is the priority; no record needs individual delivery latency.
pub fn upload_records(
    address: std::net::SocketAddr,
    records: &[[u8; 16]; 100_000],
) -> std::io::Result<usize> {
    let mut output = std::net::TcpStream::connect(address)?;
    for record in records {
        output.write_all(record)?;
    }
    output.shutdown(std::net::Shutdown::Write)?;
    Ok(records.len())
}
