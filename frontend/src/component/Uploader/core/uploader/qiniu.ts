import Chunk, { ChunkInfo } from "./chunk";
import { qiniuDriveUploadChunk, qiniuFinishUpload } from "../api";
import { Status } from "./base";

export default class Qiniu extends Chunk {
  protected async uploadChunk(chunkInfo: ChunkInfo) {
    const chunkRes = await qiniuDriveUploadChunk(
      this.task.session?.upload_urls[0]!,
      this.task.session?.credential!,
      chunkInfo,
      (p) => {
        this.updateChunkProgress(p.loaded, chunkInfo.index);
      },
      this.cancelToken.token,
    );

    this.task.chunkProgress[chunkInfo.index].etag = chunkRes.etag;
  }

  protected async afterUpload(): Promise<any> {
    this.logger.info(`Finishing multipart upload...`);
    this.transit(Status.finishing);
    return qiniuFinishUpload(
      this.task.session?.upload_urls[0]!,
      this.task.session?.credential!,
      this.task.chunkProgress,
      this.cancelToken.token,
      this.task.session?.mime_type,
    );
  }
}
