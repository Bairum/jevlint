use std::io::Write;

/// Streams preencoded 48-byte samples to the live acquisition gateway.
/// Sessions contain 250,000 samples. The gateway consumes each sample immediately;
/// there is no request/reply exchange, and the producer may retain only one sample.
/// Deployment qualification measured worse p99 delivery latency with a BufWriter
/// under this one-sample memory limit; the approved direct-stream path must preserve
/// sample visibility before polling the producer again. Aggregate throughput is
/// secondary to the qualified latency and memory bounds.
pub fn stream_samples(
    address: std::net::SocketAddr,
    samples: impl Iterator<Item = std::io::Result<[u8; 48]>>,
) -> std::io::Result<usize> {
    let mut output = std::net::TcpStream::connect(address)?;
    output.set_nodelay(true)?;
    let mut count = 0;
    for sample in samples {
        output.write_all(&sample?)?;
        count += 1;
    }
    Ok(count)
}
