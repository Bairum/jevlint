/// A constant message reader with a nonempty word slice and an in-bounds root index.
/// Word zero encodes the index of the root value in that same slice.
pub struct ConstantReader {
    pub words: &'static [u64],
}

impl ConstantReader {
    pub fn new(words: &'static [u64]) -> Result<Self, &'static str> {
        let encoded = *words.first().ok_or("missing root")?;
        let index = usize::try_from(encoded).map_err(|_| "root index too large")?;
        if index >= words.len() {
            return Err("root outside message");
        }
        Ok(Self { words })
    }

    pub fn get(&self) -> u64 {
        // SAFETY: the reader's word slice contains its encoded root index.
        unsafe {
            let index = *self.words.get_unchecked(0) as usize;
            *self.words.get_unchecked(index)
        }
    }
}

pub fn load_generated_constant() -> u64 {
    static WORDS: [u64; 2] = [1, 42];
    ConstantReader::new(&WORDS).expect("generated constant").get()
}

pub fn read_supplied_words() -> u64 {
    static WORDS: [u64; 1] = [u64::MAX];
    let reader = ConstantReader { words: &WORDS };
    reader.get()
}
