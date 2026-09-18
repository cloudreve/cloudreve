import Chunk, { ChunkInfo } from "./chunk";
import { finishOneDriveUpload, oneDriveUploadChunk } from "../api";
import { OneDriveChunkError, OneDriveEmptyFileSelected } from "../errors";
import { Status } from "./base";

export default class OneDrive extends Chunk {
  protected async uploadChunk(chunkInfo: ChunkInfo) {
    if (chunkInfo.chunk.size === 0) {
      throw new OneDriveEmptyFileSelected();
    }

    const rangeEnd = (this.progress?.total.loaded ?? 0) + chunkInfo.chunk.size - 1;
    return this.sendRange(chunkInfo, this.progress?.total.loaded ?? 0, rangeEnd, 0).catch((e) => {
      if (
        e instanceof OneDriveChunkError &&
        e.response.error.innererror &&
        e.response.error.innererror.code == "fragmentOverlap"
      ) {
        return this.alignChunkOffset(chunkInfo);
      }

      throw e;
    });
  }

  private async sendRange(chunkInfo: ChunkInfo, start: number, end: number, chunkOffset: number) {
    const range = `bytes ${start}-${end}/${this.task.file.size}`;
    return oneDriveUploadChunk(
      `${this.task.session?.upload_urls[0]!}`,
      range,
      chunkInfo,
      (p) => {
        this.updateChunkProgress(chunkOffset + p.loaded, chunkInfo.index);
      },
      this.cancelToken.token,
    );
  }

  private async alignChunkOffset(chunkInfo: ChunkInfo) {
    this.logger.info(`Chunk [${chunkInfo.index}] overlapped, checking next expected range...`);
    const rangeStatus = await oneDriveUploadChunk(
      `${this.task.session?.upload_urls[0]!}`,
      "",
      chunkInfo,
      (_p) => {
        return null;
      },
      this.cancelToken.token,
    );
    const expectedStart = parseInt(rangeStatus.nextExpectedRanges[0].split("-")[0]);
    this.logger.info(`Next expected range start from OneDrive is ${expectedStart}.`);

    const loaded = this.progress?.total.loaded ?? 0;

    if (expectedStart >= loaded) {
      this.logger.info(`This whole chunk is overlapped, skipping...`);
      this.updateChunkProgress(chunkInfo.chunk.size, chunkInfo.index);
      return;
    } else {
      this.updateChunkProgress(0, chunkInfo.index);
      const rangeEnd = loaded + chunkInfo.chunk.size - 1;
      const newChunkOffset = expectedStart - loaded;
      chunkInfo.chunk = chunkInfo.chunk.slice(newChunkOffset);
      this.updateChunkProgress(newChunkOffset, chunkInfo.index);
      return this.sendRange(chunkInfo, expectedStart, rangeEnd, newChunkOffset);
    }
  }

  protected async afterUpload(): Promise<any> {
    this.logger.info(`Finishing upload...`);
    this.transit(Status.finishing);
    return finishOneDriveUpload(this.task.session!.session_id, this.task.session!.callback_secret);
  }
}
