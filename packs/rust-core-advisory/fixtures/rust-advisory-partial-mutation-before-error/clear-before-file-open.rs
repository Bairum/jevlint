use std::fs::File;
use std::io::{self, Read};
use std::path::Path;

pub fn replace_payload(payload: &mut Vec<u8>, path: &Path) -> io::Result<()> {
    payload.clear();
    let mut file = File::open(path)?;
    let mut chunk = [0_u8; 4096];
    let size = file.read(&mut chunk)?;
    payload.extend_from_slice(&chunk[..size]);
    Ok(())
}
