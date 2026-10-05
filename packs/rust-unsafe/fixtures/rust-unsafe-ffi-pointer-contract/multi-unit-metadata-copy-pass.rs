use std::ffi::{c_char, CStr, CString};
use std::mem::{align_of, size_of};
use std::ptr;
use std::slice;

#[repr(C)]
struct RawMetadata {
    name: *const c_char,
    description: *const c_char,
    measurements: *const u32,
    measurement_count: usize,
    revision: u32,
    flags: u32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MetadataEntry {
    pub index: u32,
    pub name: String,
    pub description: Option<String>,
    pub measurements: Vec<u32>,
    pub measurement_sum: u64,
    pub revision: u32,
    pub flags: u32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum MetadataError {
    CountUnavailable,
    EntryUnavailable(u32),
    MissingName(u32),
    InvalidBuffer(u32),
}

extern "C" {
    fn metadata_count(out: *mut u32) -> i32;
    fn metadata_read(index: u32, out: *mut RawMetadata) -> i32;
    fn metadata_build_name() -> *const c_char;
}

fn empty_metadata() -> RawMetadata {
    RawMetadata {
        name: ptr::null(),
        description: ptr::null(),
        measurements: ptr::null(),
        measurement_count: 0,
        revision: 0,
        flags: 0,
    }
}

pub fn available_count() -> Result<u32, MetadataError> {
    // metadata_count writes one initialized u32 to writable out on status zero.
    // Nonzero status leaves out unchanged. The call never retains the out pointer.
    let mut count = 0_u32;
    let status = unsafe { metadata_count(&mut count) };
    if status != 0 {
        return Err(MetadataError::CountUnavailable);
    }
    Ok(count)
}

pub fn build_name() -> Option<String> {
    // metadata_build_name returns null or an immutable, NUL-terminated string
    // within one allocation of at most isize::MAX bytes, live for the process.
    // The library owns the allocation; callers must not release it.
    let name = unsafe { metadata_build_name() };
    if name.is_null() {
        return None;
    }
    Some(unsafe { CStr::from_ptr(name) }.to_string_lossy().into_owned())
}

pub fn read_metadata(limit: u32) -> Result<Vec<MetadataEntry>, MetadataError> {
    // metadata_count writes an initialized u32 on status zero and otherwise leaves
    // out unchanged. It requires a writable aligned out and never retains it.
    let mut count = 0_u32;
    let count_status = unsafe { metadata_count(&mut count) };
    if count_status != 0 {
        return Err(MetadataError::CountUnavailable);
    }
    let selected = count.min(limit);
    let mut entries = Vec::new();

    for index in 0..selected {
        // metadata_read requires writable aligned RawMetadata storage, retains
        // no out pointer, and initializes every field only on status zero.
        // A nonzero status leaves out unchanged and provides no usable record.
        // Successful names are non-null, immutable NUL-terminated strings in
        // single allocations of at most isize::MAX bytes. Description is either
        // null (absent) or has the same string guarantees.
        // All returned allocations stay immutable and live for the process;
        // subsequent metadata calls do not invalidate them. The library owns
        // them, so neither the pointers nor their buffers may be freed by Rust.
        // Measurements, when nonempty, address measurement_count initialized
        // u32 values in one allocation; the pointer is u32-aligned and the byte
        // extent is at most isize::MAX. Empty buffers may use a null pointer.
        let mut raw = empty_metadata();
        let status = unsafe { metadata_read(index, &mut raw) };
        if status != 0 {
            return Err(MetadataError::EntryUnavailable(index));
        }
        if raw.name.is_null() {
            return Err(MetadataError::MissingName(index));
        }

        let (name_ptr, _name_owner) = {
            let copied_name: CString = unsafe { CStr::from_ptr(raw.name) }.to_owned();
            (copied_name.as_ptr(), copied_name)
        };

        let description = if raw.description.is_null() {
            None
        } else {
            let borrowed_description = unsafe { CStr::from_ptr(raw.description) };
            Some(borrowed_description.to_string_lossy().into_owned())
        };

        let byte_len = raw.measurement_count.checked_mul(size_of::<u32>());
        match byte_len {
            Some(length) if length <= isize::MAX as usize => {}
            _ => return Err(MetadataError::InvalidBuffer(index)),
        }
        let borrowed_measurements: &[u32] = if raw.measurement_count == 0 {
            &[]
        } else {
            if raw.measurements.is_null()
                || (raw.measurements as usize) % align_of::<u32>() != 0
            {
                return Err(MetadataError::InvalidBuffer(index));
            }
            unsafe { slice::from_raw_parts(raw.measurements, raw.measurement_count) }
        };
        let measurement_sum = borrowed_measurements
            .iter()
            .fold(0_u64, |sum, &value| sum.saturating_add(u64::from(value)));
        let measurements = borrowed_measurements.to_vec();
        let revision = raw.revision;
        let flags = raw.flags;

        let name = unsafe { CStr::from_ptr(name_ptr) }.to_string_lossy().into_owned();
        entries.push(MetadataEntry {
            index,
            name,
            description,
            measurements,
            measurement_sum,
            revision,
            flags,
        });
    }
    Ok(entries)
}

pub fn find_by_name<'a>(entries: &'a [MetadataEntry], name: &str) -> Option<&'a MetadataEntry> {
    entries.iter().find(|entry| entry.name == name)
}

pub fn highest_revision(entries: &[MetadataEntry]) -> Option<u32> {
    entries.iter().map(|entry| entry.revision).max()
}

pub fn flagged_indices(entries: &[MetadataEntry], mask: u32) -> Vec<u32> {
    entries
        .iter()
        .filter(|entry| entry.flags & mask != 0)
        .map(|entry| entry.index)
        .collect()
}
