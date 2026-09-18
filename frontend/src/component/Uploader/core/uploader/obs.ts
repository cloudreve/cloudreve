import Chunk, { ChunkInfo } from "./chunk";
import { obsFinishUpload, s3LikeUploadChunk } from "../api";
import { Status } from "./base";

export default class OBS extends Chunk {
  protected async uploadChunk(chunkInfo: ChunkInfo) {
    const etag = await s3LikeUploadChunk(
      this.task.session?.upload_urls[chunkInfo.index]!,
      chunkInfo,
      (p) => {
        this.updateChunkProgress(p.loaded, chunkInfo.index);
      },
      this.cancelToken.token,
    );

    this.task.chunkProgress[chunkInfo.index].etag = etag;
  }

  protected async afterUpload(): Promise<any> {
    this.logger.info(`Finishing multipart upload...`);
    this.transit(Status.finishing);
    return obsFinishUpload(this.task.session!.completeURL, this.task.chunkProgress, this.cancelToken.token);
  }
}
