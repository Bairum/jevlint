use std::ffi::{c_char, CStr};

#[repr(C)]
pub struct Registry {
    _private: [u8; 0],
}

#[repr(C)]
pub struct RawEntry {
    name: *const c_char,
    values: *const u32,
    count: usize,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Entry {
    pub id: u32,
    pub name: String,
    pub values: Vec<u32>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ReadError {
    Unavailable,
    Missing(u32),
}

extern "C" {
    fn registry_open() -> *mut Registry;
    fn registry_entry(registry: *mut Registry, id: u32, out: *mut RawEntry) -> bool;
    fn registry_close(registry: *mut Registry);
}

fn empty_entry() -> RawEntry {
    RawEntry {
        name: std::ptr::null(),
        values: std::ptr::null(),
        count: 0,
    }
}

pub fn load_name(id: u32) -> Result<String, ReadError> {
    // registry_open returns null or a unique handle requiring exactly one registry_close.
    // registry_entry initializes out on success. name is non-null, NUL-terminated,
    // immutable and readable until registry_close releases all entry allocations.
    let registry = unsafe { registry_open() };
    if registry.is_null() {
        return Err(ReadError::Unavailable);
    }
    let mut raw = empty_entry();
    if !unsafe { registry_entry(registry, id, &mut raw) } {
        unsafe { registry_close(registry) };
        return Err(ReadError::Missing(id));
    }
    unsafe { registry_close(registry) };
    let name = unsafe { CStr::from_ptr(raw.name) }.to_string_lossy().into_owned();
    Ok(name)
}

pub fn load_entry(id: u32) -> Result<Entry, ReadError> {
    // registry_open returns null or a unique handle requiring exactly one registry_close.
    // registry_entry initializes out on success. name is non-null and NUL-terminated;
    // values is non-null, u32-aligned and addresses count initialized values in one
    // allocation, whose size is <= isize::MAX. Both buffers are immutable and live until
    // registry_close releases them. Empty value buffers still have a non-null pointer.
    let registry = unsafe { registry_open() };
    if registry.is_null() {
        return Err(ReadError::Unavailable);
    }
    let mut raw = empty_entry();
    if !unsafe { registry_entry(registry, id, &mut raw) } {
        unsafe { registry_close(registry) };
        return Err(ReadError::Missing(id));
    }
    let name = unsafe { CStr::from_ptr(raw.name) }.to_string_lossy().into_owned();
    let values = unsafe { std::slice::from_raw_parts(raw.values, raw.count) }.to_vec();
    unsafe { registry_close(registry) };
    Ok(Entry { id, name, values })
}

pub fn load_entries(ids: &[u32]) -> Result<Vec<Entry>, ReadError> {
    let mut entries = Vec::with_capacity(ids.len());
    for &id in ids {
        entries.push(load_entry(id)?);
    }
    Ok(entries)
}

pub fn names(entries: &[Entry]) -> Vec<&str> {
    entries.iter().map(|entry| entry.name.as_str()).collect()
}

pub fn total_values(entries: &[Entry]) -> usize {
    entries.iter().map(|entry| entry.values.len()).sum()
}

pub fn find_entry(entries: &[Entry], id: u32) -> Option<&Entry> {
    entries.iter().find(|entry| entry.id == id)
}
