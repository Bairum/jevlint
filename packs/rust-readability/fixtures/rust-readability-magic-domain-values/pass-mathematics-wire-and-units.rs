use std::time::Duration;

// Completion protocol status octet: 0x21 means accepted, 0x22 means rejected.
#[repr(u8)]
#[derive(Clone, Copy, Debug)]
pub enum CompletionStatus {
    Accepted = 0x21,
    Rejected = 0x22,
}

pub fn encode_completion(
    status: CompletionStatus,
    acknowledged: bool,
    sequence: u32,
    elapsed: Duration,
) -> [u8; 10] {
    // Wire layout: byte 0 status, byte 1 Boolean ack (0/1), bytes 2..6
    // big-endian sequence, bytes 6..10 big-endian elapsed milliseconds.
    let mut frame = [0u8; 10];
    frame[0] = status as u8;
    frame[1] = if acknowledged { 1 } else { 0 };
    frame[2..6].copy_from_slice(&sequence.to_be_bytes());
    let elapsed_ms = elapsed.as_millis().min(u32::MAX as u128) as u32;
    frame[6..10].copy_from_slice(&elapsed_ms.to_be_bytes());
    frame
}

pub fn quadratic_cost(x: f64, a: f64, b: f64, c: f64) -> (f64, f64, f64) {
    // f(x) = a*x^2 + b*x + c; derivatives are 2*a*x + b and 2*a.
    (a * x * x + b * x + c, 2.0 * a * x + b, 2.0 * a)
}

pub fn kilobytes_to_bytes(kilobytes: u64) -> Option<u64> {
    // Decimal SI kilobytes, not binary kibibytes.
    kilobytes.checked_mul(1_000)
}

pub fn kibibytes_to_bytes(kibibytes: u64) -> Option<u64> {
    kibibytes.checked_mul(1_024)
}

pub fn seconds_to_milliseconds(seconds: u64) -> Option<u64> {
    seconds.checked_mul(1_000)
}
