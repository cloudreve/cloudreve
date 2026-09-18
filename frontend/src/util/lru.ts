// LRU Cache implementation for HEIC image conversion
export class LRUCache<K, V> {
  private capacity: number;
  private cache: Map<K, V>;

  constructor(capacity: number) {
    this.capacity = capacity;
    this.cache = new Map();
  }

  get(key: K): V | undefined {
    if (this.cache.has(key)) {
      // Move to end (most recently used)
      const value = this.cache.get(key)!;
      this.cache.delete(key);
      this.cache.set(key, value);
      return value;
    }
    return undefined;
  }

  set(key: K, value: V): void {
    if (this.cache.has(key)) {
      // Update existing key
      this.cache.delete(key);
    } else if (this.cache.size >= this.capacity) {
      // Remove least recently used (first item)
      const firstKey = this.cache.keys().next().value;
      if (firstKey !== undefined) {
        const firstValue = this.cache.get(firstKey);

        // Clean up blob URL if it exists
        if (typeof firstValue === "string" && firstValue.startsWith("blob:")) {
          URL.revokeObjectURL(firstValue);
        }

        this.cache.delete(firstKey);
      }
    }
    this.cache.set(key, value);
  }

  clear(): void {
    // Clean up all blob URLs
    for (const value of this.cache.values()) {
      if (typeof value === "string" && value.startsWith("blob:")) {
        URL.revokeObjectURL(value);
      }
    }
    this.cache.clear();
  }
}
