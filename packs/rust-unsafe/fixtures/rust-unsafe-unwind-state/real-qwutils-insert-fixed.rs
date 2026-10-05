pub struct Record {
    pub payload: Box<u8>,
    pub fail_clone: bool,
}

impl Clone for Record {
    fn clone(&self) -> Self {
        assert!(!self.fail_clone, "clone rejected");
        Self { payload: self.payload.clone(), fail_clone: self.fail_clone }
    }
}

pub fn insert_cloned(values: &mut Vec<Record>, source: &Record) {
    assert_eq!(values.len(), 2);
    values.reserve(1);
    let pointer = values.as_mut_ptr();
    unsafe {
        values.set_len(0);
        std::ptr::copy(pointer, pointer.add(1), 2);
    }
    let inserted = source.clone();
    unsafe {
        std::ptr::write(pointer, inserted);
        values.set_len(3);
    }
}
