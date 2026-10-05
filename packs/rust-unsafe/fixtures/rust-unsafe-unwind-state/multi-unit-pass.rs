pub struct Batch {
    pub sequence: u32,
    pub values: Vec<Box<u8>>,
}

pub struct BatchSummary {
    pub sequence: u32,
    pub total: u64,
    pub count: usize,
}

pub fn create_batch(sequence: u32, bytes: &[u8]) -> Batch {
    let mut values = Vec::with_capacity(bytes.len());
    for byte in bytes {
        values.push(Box::new(*byte));
    }
    Batch { sequence, values }
}

pub fn summarize(batch: &Batch) -> BatchSummary {
    let mut total = 0_u64;
    for value in &batch.values {
        total += u64::from(**value);
    }
    BatchSummary {
        sequence: batch.sequence,
        total,
        count: batch.values.len(),
    }
}

pub fn inspect_last<F: FnOnce(&u8)>(batch: &mut Batch, callback: F) {
    let Some(index) = batch.values.len().checked_sub(1) else {
        return;
    };
    let pointer = unsafe { batch.values.as_mut_ptr().add(index) };
    unsafe { batch.values.set_len(index) };
    let item = unsafe { std::ptr::read(pointer) };
    callback(item.as_ref());
    batch.values.push(item);
}

pub fn encode(batch: &Batch) -> Vec<u8> {
    let mut bytes = Vec::new();
    bytes.extend_from_slice(&batch.sequence.to_le_bytes());
    for value in &batch.values {
        bytes.push(**value);
    }
    bytes
}

pub fn clamp(batch: &mut Batch, maximum: u8) {
    for value in &mut batch.values {
        **value = (**value).min(maximum);
    }
}

pub fn remove_zeroes(batch: &mut Batch) {
    batch.values.retain(|value| **value != 0);
}

pub fn append(batch: &mut Batch, bytes: &[u8]) {
    batch.values.reserve(bytes.len());
    for byte in bytes {
        batch.values.push(Box::new(*byte));
    }
}

pub fn take_above(batch: &mut Batch, threshold: u8) -> Vec<Box<u8>> {
    let mut remaining = Vec::new();
    let mut selected = Vec::new();
    for value in batch.values.drain(..) {
        if *value > threshold {
            selected.push(value);
        } else {
            remaining.push(value);
        }
    }
    batch.values = remaining;
    selected
}

pub fn merge(sequence: u32, batches: Vec<Batch>) -> Batch {
    let mut values = Vec::new();
    for mut batch in batches {
        values.append(&mut batch.values);
    }
    Batch { sequence, values }
}

pub fn decode(bytes: &[u8]) -> Option<Batch> {
    let header = bytes.get(..4)?;
    let sequence = u32::from_le_bytes([header[0], header[1], header[2], header[3]]);
    Some(create_batch(sequence, bytes.get(4..)?))
}
