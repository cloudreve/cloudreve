use base64::{engine::general_purpose::STANDARD, Engine};

/// A compact boolean set stored as a bit array
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Boolset {
    data: Vec<u8>,
}

impl Boolset {
    /// Create a new empty Boolset
    pub fn new() -> Self {
        Self { data: Vec::new() }
    }

    /// Create a Boolset from raw bytes
    pub fn from_raw(data: Vec<u8>) -> Self {
        Self { data }
    }

    /// Create a Boolset from a base64-encoded string
    pub fn from_base64(encoded: &str) -> Result<Self, base64::DecodeError> {
        let data = STANDARD.decode(encoded)?;
        Ok(Self { data })
    }

    /// Create a Boolset from an optional base64 string, falling back to raw bytes or empty
    /// This mimics the TypeScript constructor behavior
    pub fn from_data(base64_str: Option<&str>, raw: Option<Vec<u8>>) -> Self {
        if let Some(encoded) = base64_str {
            Self::from_base64(encoded).unwrap_or_else(|e| {
                eprintln!("Failed to decode boolset: {}", e);
                Self::new()
            })
        } else if let Some(data) = raw {
            Self::from_raw(data)
        } else {
            Self::new()
        }
    }

    /// Check if a bit at the given index is enabled
    pub fn enabled(&self, index: usize) -> bool {
        if index >= self.data.len() * 8 {
            return false;
        }
        (self.data[index / 8] & (1 << (index % 8))) != 0
    }

    /// Perform bitwise AND with another Boolset, returning a new Boolset
    pub fn and(&self, other: &Boolset) -> Boolset {
        let length = self.data.len().max(other.data.len());
        let mut result = vec![0u8; length];

        for i in 0..length {
            let a = self.data.get(i).copied().unwrap_or(0);
            let b = other.data.get(i).copied().unwrap_or(0);
            result[i] = a & b;
        }

        Boolset { data: result }
    }

    /// Perform bitwise OR with another Boolset, returning a new Boolset
    pub fn or(&self, other: &Boolset) -> Boolset {
        let length = self.data.len().max(other.data.len());
        let mut result = vec![0u8; length];

        for i in 0..length {
            let a = self.data.get(i).copied().unwrap_or(0);
            let b = other.data.get(i).copied().unwrap_or(0);
            result[i] = a | b;
        }

        Boolset { data: result }
    }

    /// Set or clear a bit at the given index
    /// Returns a mutable reference to self for method chaining
    pub fn set(&mut self, index: usize, enabled: bool) -> &mut Self {
        let byte_index = index / 8;
        let bit_index = index % 8;

        // Expand array if necessary
        if byte_index >= self.data.len() {
            self.data.resize(byte_index + 1, 0);
        }

        if enabled {
            self.data[byte_index] |= 1 << bit_index;
        } else {
            self.data[byte_index] &= !(1 << bit_index);
        }

        self
    }

    /// Set multiple bits at once from a slice of (index, enabled) tuples
    pub fn sets(&mut self, values: &[(usize, bool)]) -> &mut Self {
        for &(index, enabled) in values {
            self.set(index, enabled);
        }
        self
    }

    /// Convert to base64-encoded string
    pub fn to_base64(&self) -> String {
        STANDARD.encode(&self.data)
    }

    /// Get the underlying byte data
    pub fn as_bytes(&self) -> &[u8] {
        &self.data
    }
}

impl Default for Boolset {
    fn default() -> Self {
        Self::new()
    }
}
