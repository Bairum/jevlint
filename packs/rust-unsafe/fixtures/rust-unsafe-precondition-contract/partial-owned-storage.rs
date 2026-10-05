pub fn stage_names(names: &[&str]) -> ([std::mem::MaybeUninit<String>; 4], usize) {
    let mut storage: [std::mem::MaybeUninit<String>; 4] =
        std::array::from_fn(|_| std::mem::MaybeUninit::uninit());
    let mut written = 0;
    for name in names.iter().take(storage.len()) {
        unsafe { storage.as_mut_ptr().add(written).write(std::mem::MaybeUninit::new((*name).to_owned())) };
        written += 1;
    }
    (storage, written)
}
