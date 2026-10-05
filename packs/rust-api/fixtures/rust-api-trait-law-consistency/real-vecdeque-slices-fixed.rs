use std::hash::{Hash, Hasher};

/// A byte queue whose logical contents are the leading segment followed by
/// the trailing segment. Segment boundaries are storage details.
pub struct ByteQueue {
    pub leading: Vec<u8>,
    pub trailing: Vec<u8>,
}

impl ByteQueue {
    pub fn len(&self) -> usize {
        self.leading.len() + self.trailing.len()
    }

    pub fn is_empty(&self) -> bool {
        self.leading.is_empty() && self.trailing.is_empty()
    }

    pub fn bytes(&self) -> impl Iterator<Item = &u8> {
        self.leading.iter().chain(self.trailing.iter())
    }
}

impl PartialEq for ByteQueue {
    fn eq(&self, other: &Self) -> bool {
        self.bytes().eq(other.bytes())
    }
}

impl Eq for ByteQueue {}

impl Hash for ByteQueue {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.len().hash(state);
        for byte in self.bytes() {
            byte.hash(state);
        }
    }
}

pub fn queue_from_segments(leading: &[u8], trailing: &[u8]) -> ByteQueue {
    ByteQueue {
        leading: leading.to_vec(),
        trailing: trailing.to_vec(),
    }
}
