use std::fs::{self, OpenOptions};
use std::io::{self, Write};
use std::path::{Path, PathBuf};

const FRAME_MAGIC: &[u8; 4] = b"TLM1";
const CRC32_POLYNOMIAL: u32 = 0xedb8_8320;

pub struct Sample {
    pub channel: u16,
    pub observed_ms: u64,
    pub value: i32,
}

pub struct Receipt {
    pub path: PathBuf,
    pub sample_count: usize,
    pub encoded_bytes: usize,
}

pub struct TelemetrySpool {
    directory: PathBuf,
    next_sequence: u64,
}

impl TelemetrySpool {
    pub fn new(directory: PathBuf, next_sequence: u64) -> Self {
        Self { directory, next_sequence }
    }

    pub fn publish(&mut self, samples: &[Sample]) -> io::Result<Receipt> {
        if samples.is_empty() {
            return Err(io::Error::new(io::ErrorKind::InvalidInput, "empty batch"));
        }
        let sequence = self.next_sequence;
        let next_sequence = sequence.checked_add(1).ok_or_else(|| {
            io::Error::new(io::ErrorKind::InvalidInput, "sequence exhausted")
        })?;
        fs::create_dir_all(&self.directory)?;
        let frame = encode_batch(sequence, samples)?;
        let path = self.directory.join(format!("batch-{sequence:020}.bin"));
        persist_frame(&path, &frame)?;
        self.next_sequence = next_sequence;
        Ok(Receipt {
            path,
            sample_count: samples.len(),
            encoded_bytes: frame.len(),
        })
    }
}

fn encode_batch(sequence: u64, samples: &[Sample]) -> io::Result<Vec<u8>> {
    let sample_count = u32::try_from(samples.len()).map_err(|_| {
        io::Error::new(io::ErrorKind::InvalidInput, "too many samples")
    })?;
    let base_ms = samples.iter().map(|sample| sample.observed_ms).min().unwrap_or(0);
    let mut frame = Vec::new();
    frame.extend_from_slice(FRAME_MAGIC);
    frame.extend_from_slice(&sequence.to_le_bytes());
    frame.extend_from_slice(&base_ms.to_le_bytes());
    frame.extend_from_slice(&sample_count.to_le_bytes());
    for sample in samples {
        frame.extend_from_slice(&sample.channel.to_le_bytes());
        let delta_ms = u32::try_from(sample.observed_ms - base_ms).map_err(|_| {
            io::Error::new(io::ErrorKind::InvalidInput, "batch time span too large")
        })?;
        let zigzag_value = ((sample.value as u32) << 1) ^ ((sample.value >> 31) as u32);
        for mut value in [delta_ms, zigzag_value] {
            loop {
                let mut byte = (value & 0x7f) as u8;
                value >>= 7;
                if value != 0 {
                    byte |= 0x80;
                }
                frame.push(byte);
                if value == 0 {
                    break;
                }
            }
        }
    }
    let mut checksum = u32::MAX;
    for &byte in &frame {
        checksum ^= u32::from(byte);
        for _ in 0..8 {
            checksum = if checksum & 1 != 0 {
                (checksum >> 1) ^ CRC32_POLYNOMIAL
            } else {
                checksum >> 1
            };
        }
    }
    frame.extend_from_slice(&(!checksum).to_le_bytes());
    Ok(frame)
}

fn persist_frame(path: &Path, frame: &[u8]) -> io::Result<()> {
    let mut file = OpenOptions::new().write(true).create_new(true).open(path)?;
    file.write_all(frame)?;
    file.sync_data()
}
