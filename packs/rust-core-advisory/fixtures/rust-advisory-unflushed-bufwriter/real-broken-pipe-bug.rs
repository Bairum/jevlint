use std::io::{self, BufWriter, Write};
use std::net::{SocketAddr, TcpStream};
use std::time::Duration;

pub struct Metric {
    pub name: String,
    pub value: f64,
    pub timestamp: u64,
}

pub struct Batch {
    pub source: String,
    pub sequence: u64,
    pub metrics: Vec<Metric>,
}

fn validate_name(name: &str) -> io::Result<()> {
    if name.is_empty() || name.len() > 128 {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid name length"));
    }
    if !name.bytes().all(|byte| {
        byte.is_ascii_alphanumeric() || matches!(byte, b'.' | b'_' | b'-')
    }) {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid name character"));
    }
    Ok(())
}

fn validate_batch(batch: &Batch) -> io::Result<()> {
    validate_name(&batch.source)?;
    if batch.metrics.is_empty() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "empty batch"));
    }
    for metric in &batch.metrics {
        validate_name(&metric.name)?;
        if !metric.value.is_finite() {
            return Err(io::Error::new(io::ErrorKind::InvalidInput, "nonfinite value"));
        }
    }
    Ok(())
}

fn connect_writer(address: SocketAddr, timeout: Duration) -> io::Result<BufWriter<TcpStream>> {
    let stream = TcpStream::connect_timeout(&address, timeout)?;
    stream.set_write_timeout(Some(timeout))?;
    Ok(BufWriter::new(stream))
}

fn write_header(writer: &mut BufWriter<TcpStream>, batch: &Batch) -> io::Result<()> {
    writeln!(writer, "METRICS/1")?;
    writeln!(writer, "source {}", batch.source)?;
    writeln!(writer, "sequence {}", batch.sequence)?;
    writeln!(writer, "count {}", batch.metrics.len())
}

fn write_metric(writer: &mut BufWriter<TcpStream>, metric: &Metric) -> io::Result<()> {
    writeln!(writer, "{} {} {}", metric.name, metric.value, metric.timestamp)
}

pub fn send_batch(
    address: SocketAddr,
    timeout: Duration,
    batch: &Batch,
) -> io::Result<usize> {
    validate_batch(batch)?;
    let mut writer: BufWriter<TcpStream> = connect_writer(address, timeout)?;
    write_header(&mut writer, batch)?;
    for metric in &batch.metrics {
        write_metric(&mut writer, metric)?;
    }
    writeln!(writer, "END")?;
    Ok(batch.metrics.len())
}

pub fn send_measurement(
    address: SocketAddr,
    source: &str,
    sequence: u64,
    metric: Metric,
) -> io::Result<usize> {
    let batch = Batch {
        source: source.to_owned(),
        sequence,
        metrics: vec![metric],
    };
    send_batch(address, Duration::from_secs(5), &batch)
}
