import { s3LikeFinishUpload, s3LikeUploadChunk } from "../api";
import { Status } from "./base";
import Chunk, { ChunkInfo } from "./chunk";

export default class OSS extends Chunk {
  protected async uploadChunk(chunkInfo: ChunkInfo) {
    return s3LikeUploadChunk(
      this.task.session?.upload_urls[chunkInfo.index]!,
      chunkInfo,
      (p) => {
        this.updateChunkProgress(p.loaded, chunkInfo.index);
      },
      this.cancelToken.token,
    );
  }

  protected async afterUpload(): Promise<any> {
    this.logger.info(`Finishing multipart upload...`);
    this.transit(Status.finishing);
    return s3LikeFinishUpload(this.task.session!.completeURL, true, this.task.chunkProgress, this.cancelToken.token, {
      "x-oss-forbid-overwrite": "true",
      "x-oss-complete-all": "yes",
    });
  }
}
