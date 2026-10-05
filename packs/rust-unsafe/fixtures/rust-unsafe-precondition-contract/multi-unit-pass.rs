pub struct Telemetry {
    pub sequence: u32,
    pub channel: u16,
    pub samples: Vec<u16>,
}

pub struct Summary {
    pub minimum: u16,
    pub maximum: u16,
    pub total: u64,
    pub count: usize,
}

pub fn parse_samples(bytes: &[u8]) -> Option<Vec<u16>> {
    if bytes.len() % 2 != 0 {
        return None;
    }
    let mut samples = Vec::with_capacity(bytes.len() / 2);
    for pair in bytes.chunks_exact(2) {
        samples.push(u16::from_le_bytes([pair[0], pair[1]]));
    }
    Some(samples)
}

pub fn summarize(samples: &[u16]) -> Option<Summary> {
    let first = *samples.first()?;
    let mut summary = Summary {
        minimum: first,
        maximum: first,
        total: 0,
        count: samples.len(),
    };
    for sample in samples {
        summary.minimum = summary.minimum.min(*sample);
        summary.maximum = summary.maximum.max(*sample);
        summary.total += u64::from(*sample);
    }
    Some(summary)
}

pub fn encode(frame: &Telemetry) -> Vec<u8> {
    let mut bytes = Vec::new();
    bytes.extend_from_slice(&frame.sequence.to_le_bytes());
    bytes.extend_from_slice(&frame.channel.to_le_bytes());
    for sample in &frame.samples {
        bytes.extend_from_slice(&sample.to_le_bytes());
    }
    bytes
}

pub fn tail_total(frame: &Telemetry, skipped: usize) -> Option<u64> {
    if skipped > frame.samples.len() {
        return None;
    }
    let remaining = frame.samples.len() - skipped;
    let pointer = unsafe { frame.samples.as_ptr().add(skipped) };
    let tail = unsafe { std::slice::from_raw_parts(pointer, remaining) };
    Some(tail.iter().map(|sample| u64::from(*sample)).sum())
}

pub fn clamp_samples(frame: &mut Telemetry, maximum: u16) {
    for sample in &mut frame.samples {
        *sample = (*sample).min(maximum);
    }
}

pub fn split_channels(frames: Vec<Telemetry>, channel: u16) -> (Vec<Telemetry>, Vec<Telemetry>) {
    let mut selected = Vec::new();
    let mut other = Vec::new();
    for frame in frames {
        if frame.channel == channel {
            selected.push(frame);
        } else {
            other.push(frame);
        }
    }
    (selected, other)
}

pub fn merge_samples(frames: &[Telemetry], channel: u16) -> Vec<u16> {
    let mut samples = Vec::new();
    for frame in frames {
        if frame.channel == channel {
            samples.extend_from_slice(&frame.samples);
        }
    }
    samples
}

pub fn decode(sequence: u32, channel: u16, bytes: &[u8]) -> Option<Telemetry> {
    Some(Telemetry {
        sequence,
        channel,
        samples: parse_samples(bytes)?,
    })
}
