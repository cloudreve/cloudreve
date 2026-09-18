import * as utils from "../utils";
import Base from "./base";
import { EncryptedBlob } from "./encrypt/blob";

export interface ChunkProgress {
  loaded: number;
  index: number;
  etag?: string;
}

export interface ChunkInfo {
  chunk: Blob;
  index: number;
}

export default abstract class Chunk extends Base {
  protected chunks: Blob[] = [];
  private readonly DEFAULT_CONCURRENCY = 1; // Default concurrent uploads
  private readonly MAX_RETRIES = 3;
  private progressUpdateMutex = Promise.resolve(); // Ensure progress updates are serialized

  public upload = async () => {
    this.logger.info("Preparing uploading file chunks.");
    this.initBeforeUploadChunks();

    this.logger.info("Starting concurrent uploading of file chunks:", this.chunks);
    this.updateLocalCache();

    await this.uploadChunksWithDynamicPool();
  };

  private async uploadChunksWithDynamicPool() {
    // Get chunks that need to be uploaded
    const chunksToUpload = this.chunks
      .map((chunk, index) => ({ chunk, index }))
      .filter(({ chunk, index }) => this.task.chunkProgress[index].loaded < chunk.size || chunk.size === 0);

    if (chunksToUpload.length === 0) {
      this.logger.info("All chunks already uploaded, skipping.");
      return;
    }

    this.logger.info(`Found ${chunksToUpload.length} chunks to upload out of ${this.chunks.length} total chunks.`);

    const concurrency = this.getConcurrency();
    let chunkIndex = 0;
    let activeCount = 0;
    let hasError = false;
    let firstError: any = null;

    // Helper function to start a new upload with immediate callback
    const startUpload = (chunkInfo: { chunk: Blob; index: number }): Promise<void> => {
      // Don't start new uploads if there's already an error
      if (hasError) {
        return Promise.resolve();
      }

      activeCount++;
      this.logger.info(`Starting chunk [${chunkInfo.index}], active uploads: ${activeCount}`);

      return this.uploadChunkWithRetryAndCallback(chunkInfo, () => {
        this.updateLocalCache();
      })
        .then(() => {
          activeCount--;
          this.logger.info(`Chunk [${chunkInfo.index}] completed successfully, active uploads: ${activeCount}`);

          // Start next chunk immediately if available and no error occurred
          if (!hasError && chunkIndex < chunksToUpload.length) {
            startUpload(chunksToUpload[chunkIndex++]);
          }
        })
        .catch((error) => {
          activeCount--;
          this.logger.error(`Chunk [${chunkInfo.index}] failed, stopping all uploads:`, error);

          // Mark error state to stop new uploads
          hasError = true;
          if (!firstError) {
            firstError = error;
          }

          // Cancel all remaining uploads
          this.cancelToken.cancel();
        });
    };

    // Start initial uploads up to concurrency limit
    const initialPromises: Promise<void>[] = [];
    while (chunkIndex < chunksToUpload.length && initialPromises.length < concurrency) {
      initialPromises.push(startUpload(chunksToUpload[chunkIndex++]));
    }

    // Wait for all uploads to complete or fail
    await Promise.allSettled(initialPromises);

    // Wait for any remaining uploads started by callbacks to finish
    while (activeCount > 0) {
      await new Promise((resolve) => setTimeout(resolve, 100)); // Small delay to check active count
    }

    // Throw error immediately if any chunk failed
    if (firstError) {
      this.logger.error("Upload process stopped due to chunk failure");
      throw firstError;
    }

    this.logger.info("All chunks uploaded successfully.");
  }

  private async uploadChunkWithRetryAndCallback(
    chunkInfo: ChunkInfo,
    onComplete: () => void,
    retryCount = 0,
  ): Promise<void> {
    // Check for cancellation before attempting upload
    if (this.cancelToken.token.reason) {
      throw new Error("Upload cancelled by user");
    }

    try {
      if (chunkInfo.chunk instanceof EncryptedBlob && !this.task.policy.streaming_encryption) {
        chunkInfo.chunk = new Blob([await chunkInfo.chunk.bytes()]);
      }
      await this.uploadChunk(chunkInfo);
      this.logger.info(`Chunk [${chunkInfo.index}] uploaded successfully.`);
      onComplete(); // Call callback immediately after successful upload
    } catch (error) {
      // Don't retry if upload was cancelled
      if (this.cancelToken.token.reason) {
        throw error;
      }

      if (retryCount < this.MAX_RETRIES) {
        this.logger.warn(
          `Chunk [${chunkInfo.index}] upload failed, retrying (${retryCount + 1}/${this.MAX_RETRIES}):`,
          error,
        );

        // Wait with exponential backoff, but check for cancellation
        const delay = Math.pow(2, retryCount) * 1000;
        await new Promise((resolve, reject) => {
          const timeout = setTimeout(resolve, delay);

          // Cancel the timeout if upload is cancelled
          this.cancelToken.token.promise?.then(() => {
            clearTimeout(timeout);
            reject(new Error("Upload cancelled during retry delay"));
          });
        });

        await this.uploadChunkWithRetryAndCallback(chunkInfo, onComplete, retryCount + 1);
      } else {
        this.logger.error(`Chunk [${chunkInfo.index}] upload failed after ${this.MAX_RETRIES} retries:`, error);
        throw error;
      }
    }
  }

  private getConcurrency(): number {
    return this.task.policy.chunk_concurrency || this.DEFAULT_CONCURRENCY;
  }

  private initBeforeUploadChunks() {
    this.chunks = utils.getChunks(this.task.blob, this.task.session?.chunk_size);
    const cachedInfo = utils.getResumeCtx(this.task, this.logger);
    if (cachedInfo == null) {
      this.task.chunkProgress = this.chunks.map(
        (_, index): ChunkProgress => ({
          loaded: 0,
          index,
        }),
      );
    }

    this.notifyResumeProgress();
  }

  protected abstract uploadChunk(chunkInfo: ChunkInfo): Promise<any>;

  protected updateChunkProgress(loaded: number, index: number, etag?: string) {
    // Serialize progress updates to avoid race conditions in concurrent uploads
    this.progressUpdateMutex = this.progressUpdateMutex.then(async () => {
      this.task.chunkProgress[index].loaded = loaded;
      if (etag) {
        this.task.chunkProgress[index].etag = etag;
      }
      this.notifyResumeProgress();
    });
  }

  private notifyResumeProgress() {
    this.progress = {
      total: this.getProgressInfoItem(utils.sumChunk(this.task.chunkProgress), this.task.file.size + 1),
      chunks: this.chunks.map((chunk, index) => {
        return this.getProgressInfoItem(this.task.chunkProgress[index].loaded, chunk.size, false);
      }),
    };
    this.subscriber.onProgress(this.progress);
  }

  private updateLocalCache() {
    utils.setResumeCtx(this.task, this.logger);
  }
}
