import { EncryptionCipher, EncryptMetadata } from "../../../../../api/explorer";

/**
 * EncryptedBlob wraps a Blob and encrypts its stream on-the-fly using the provided encryption metadata.
 * This allows for client-side encryption during upload without loading the entire file into memory.
 *
 * ## Counter Handling for AES-CTR Mode
 *
 * AES-CTR (Counter) mode encryption requires careful counter management:
 * - Each 16-byte block uses a unique counter value
 * - Counter increments by 1 for each block
 * - For byte position N: counter = initial_counter + floor(N / 16)
 *
 * ## Slicing Support
 *
 * When slice() is called, the new EncryptedBlob tracks the byte offset (counterOffset).
 * This ensures that:
 * 1. Block-aligned slices (offset % 16 == 0) encrypt correctly
 * 2. Non-block-aligned slices handle partial blocks by padding and extracting
 *
 * Example:
 * ```
 * const encrypted = new EncryptedBlob(file, metadata);
 * const chunk1 = encrypted.slice(0, 5MB);      // Encrypts bytes [0, 5MB) with counter starting at base
 * const chunk2 = encrypted.slice(5MB, 10MB);   // Encrypts bytes [5MB, 10MB) with counter offset by 5MB/16 blocks
 * ```
 *
 * The encrypted output of sliced chunks will match what would be produced if the entire
 * blob was encrypted as one stream, maintaining consistency for chunked uploads.
 */
export class EncryptedBlob implements Blob {
  private readonly blob: Blob;
  private readonly metadata: EncryptMetadata;
  private readonly counterOffset: number;
  private cryptoKey?: CryptoKey;

  constructor(blob: Blob, metadata: EncryptMetadata, counterOffset: number = 0) {
    this.blob = blob;
    this.metadata = metadata;
    this.counterOffset = counterOffset;
  }

  /**
   * Returns the size of the original blob.
   * Note: Encrypted size may differ depending on algorithm, but for AES-CTR it remains the same.
   */
  get size(): number {
    return this.blob.size;
  }

  get type(): string {
    return this.blob.type;
  }

  /**
   * Converts hex string or base64 string to Uint8Array
   */
  private stringToUint8Array(str: string, encoding: "hex" | "base64" = "base64"): Uint8Array<ArrayBuffer> {
    if (encoding === "hex") {
      // Remove any whitespace or separators
      const cleaned = str.replace(/[^0-9a-fA-F]/g, "");
      const buffer = new ArrayBuffer(cleaned.length / 2);
      const bytes = new Uint8Array(buffer);
      for (let i = 0; i < cleaned.length; i += 2) {
        bytes[i / 2] = parseInt(cleaned.substring(i, i + 2), 16);
      }
      return bytes;
    } else {
      // base64 decoding
      const binaryString = atob(str);
      const buffer = new ArrayBuffer(binaryString.length);
      const bytes = new Uint8Array(buffer);
      for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
      }
      return bytes;
    }
  }

  /**
   * Increment a counter (Uint8Array) by a given number of blocks
   */
  private incrementCounter(counter: Uint8Array<ArrayBuffer>, blocks: number): Uint8Array<ArrayBuffer> {
    // Create a copy to avoid modifying the original counter
    const result = new Uint8Array(counter);

    // AES-CTR uses big-endian counter increment
    // Start from the least significant byte (rightmost) and work backwards
    let carry = blocks;
    for (let i = result.length - 1; i >= 0 && carry > 0; i--) {
      const sum = result[i] + carry;
      result[i] = sum & 0xff;
      carry = sum >>> 8; // Shift right by 8 bits to propagate overflow as carry
    }

    return result;
  }

  /**
   * Import the encryption key for use with Web Crypto API
   */
  private async importKey(): Promise<CryptoKey> {
    if (this.cryptoKey) {
      return this.cryptoKey;
    }

    const keyBytes = this.stringToUint8Array(this.metadata.key_plain_text);

    switch (this.metadata.algorithm) {
      case EncryptionCipher.aes256ctr:
        this.cryptoKey = await crypto.subtle.importKey("raw", keyBytes, { name: "AES-CTR" }, false, ["encrypt"]);
        break;
      default:
        throw new Error(`Unsupported encryption algorithm: ${this.metadata.algorithm}`);
    }

    return this.cryptoKey;
  }

  /**
   * Create an encryption transform stream
   */
  private async createEncryptStream(): Promise<TransformStream<Uint8Array, Uint8Array>> {
    const cryptoKey = await this.importKey();
    const iv = this.stringToUint8Array(this.metadata.iv);

    // Create counter value (16 bytes IV) and apply offset for sliced blobs
    let counter = new Uint8Array(16);
    counter.set(iv.slice(0, 16));

    // Apply counter offset based on byte position (each block is 16 bytes)
    // For non-block-aligned offsets, we handle partial blocks correctly
    if (this.counterOffset > 0) {
      const blockOffset = Math.floor(this.counterOffset / 16);
      counter = this.incrementCounter(counter, blockOffset);
    }

    const that = this;
    // Track bytes processed to handle partial blocks correctly
    let totalBytesProcessed = this.counterOffset;
    // Remember if we've processed the first chunk (which may be non-block-aligned)
    let isFirstChunk = true;

    return new TransformStream<Uint8Array, Uint8Array>({
      async transform(chunk, controller) {
        // Create a new ArrayBuffer copy to ensure proper type for crypto API
        const buffer = new ArrayBuffer(chunk.byteLength);
        const chunkData = new Uint8Array(buffer);
        if (chunk instanceof Uint8Array) {
          chunkData.set(new Uint8Array(chunk.buffer, chunk.byteOffset, chunk.byteLength));
        } else {
          chunkData.set(new Uint8Array(chunk));
        }

        // Handle partial block at the start (only for the first chunk if offset is not block-aligned)
        const offsetInBlock = totalBytesProcessed % 16;
        if (isFirstChunk && offsetInBlock !== 0) {
          // We're starting mid-block. Need to encrypt the partial block correctly.
          // Pad to 16 bytes, encrypt, then extract only the bytes we need
          const partialBlockSize = Math.min(16 - offsetInBlock, chunkData.byteLength);
          const paddedBlock = new Uint8Array(16);
          paddedBlock.set(chunkData.slice(0, partialBlockSize), offsetInBlock);

          const encryptedBlock = await crypto.subtle.encrypt(
            { name: "AES-CTR", counter, length: 128 },
            cryptoKey,
            paddedBlock,
          );

          const encryptedBytes = new Uint8Array(encryptedBlock);
          controller.enqueue(encryptedBytes.slice(offsetInBlock, offsetInBlock + partialBlockSize));

          // Increment counter by 1 block
          counter = that.incrementCounter(counter, 1);
          totalBytesProcessed += partialBlockSize;

          // Process remaining bytes if any
          if (partialBlockSize < chunkData.byteLength) {
            const remaining = chunkData.slice(partialBlockSize);
            const encryptedRemaining = await crypto.subtle.encrypt(
              { name: "AES-CTR", counter, length: 128 },
              cryptoKey,
              remaining,
            );
            controller.enqueue(new Uint8Array(encryptedRemaining));

            // Increment counter by number of blocks processed
            const blocksProcessed = Math.ceil(remaining.byteLength / 16);
            counter = that.incrementCounter(counter, blocksProcessed);
            totalBytesProcessed += remaining.byteLength;
          }
          isFirstChunk = false;
        } else {
          // Normal case: block-aligned encryption
          const encrypted = await crypto.subtle.encrypt(
            { name: "AES-CTR", counter, length: 128 },
            cryptoKey,
            chunkData,
          );

          // Send encrypted chunk
          controller.enqueue(new Uint8Array(encrypted));

          // Update counter: increment by number of 16-byte blocks (rounded up for partial blocks)
          const blocksProcessed = Math.ceil(chunkData.byteLength / 16);
          counter = that.incrementCounter(counter, blocksProcessed);
          totalBytesProcessed += chunkData.byteLength;
          isFirstChunk = false;
        }
      },
    });
  }

  /**
   * Returns an encrypted stream of the blob's contents
   */
  stream(): ReadableStream {
    const originalStream = this.blob.stream();
    const encryptStreamPromise = this.createEncryptStream();

    // Create a passthrough stream that will pipe through the encrypt stream once it's ready
    return new ReadableStream({
      async start(controller) {
        const encryptStream = await encryptStreamPromise;
        const encryptedStream = originalStream.pipeThrough(encryptStream);
        const reader = encryptedStream.getReader();

        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            controller.enqueue(value);
          }
          controller.close();
        } catch (error) {
          controller.error(error);
        }
      },
    });
  }

  /**
   * Returns encrypted blob slice
   * The counter offset is calculated to ensure correct encryption alignment
   * for the sliced portion of the blob.
   */
  slice(start?: number, end?: number, contentType?: string): Blob {
    const slicedBlob = this.blob.slice(start, end, contentType);

    // Calculate the new counter offset
    // The offset accumulates: if this blob already has an offset, add to it
    const newOffset = this.counterOffset + (start || 0);

    return new EncryptedBlob(slicedBlob, this.metadata, newOffset);
  }

  /**
   * Returns encrypted data as ArrayBuffer
   */
  async arrayBuffer(): Promise<ArrayBuffer> {
    const stream = this.stream();
    const reader = stream.getReader();
    const chunks: Uint8Array[] = [];

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(value);
      }
    } finally {
      reader.releaseLock();
    }

    // Concatenate all chunks
    const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const chunk of chunks) {
      result.set(chunk, offset);
      offset += chunk.length;
    }

    return result.buffer;
  }

  /**
   * Returns encrypted data as text (likely not useful, but required by Blob interface)
   */
  async text(): Promise<string> {
    const buffer = await this.arrayBuffer();
    const decoder = new TextDecoder();
    return decoder.decode(buffer);
  }

  /**
   * Returns encrypted data as Uint8Array (required by Blob interface)
   */
  async bytes(): Promise<Uint8Array<ArrayBuffer>> {
    const buffer = await this.arrayBuffer();
    return new Uint8Array(buffer);
  }
}
