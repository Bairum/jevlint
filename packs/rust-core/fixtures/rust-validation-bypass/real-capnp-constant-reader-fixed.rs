/// A constant message reader with a nonempty word slice and an in-bounds root index.
/// Word zero encodes the index of the root value in that same slice.
pub struct ConstantReader {
    words: &'static [u64],
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

    /// # Safety
    /// The words must be nonempty; word zero must fit usize and index this slice.
    pub unsafe fn from_words_unchecked(words: &'static [u64]) -> Self {
        Self { words }
    }

    pub fn get(&self) -> u64 {
        // SAFETY: both constructors establish that word zero indexes this slice.
        unsafe {
            let index = *self.words.get_unchecked(0) as usize;
            *self.words.get_unchecked(index)
        }
    }
}

pub fn load_generated_constant() -> u64 {
    static WORDS: [u64; 2] = [1, 42];
    // SAFETY: word zero is 1, which fits usize and indexes the two-word slice.
    let reader = unsafe { ConstantReader::from_words_unchecked(&WORDS) };
    reader.get()
}

pub fn read_supplied_words() -> Result<u64, &'static str> {
    static WORDS: [u64; 1] = [u64::MAX];
    let reader = ConstantReader::new(&WORDS)?;
    Ok(reader.get())
}
