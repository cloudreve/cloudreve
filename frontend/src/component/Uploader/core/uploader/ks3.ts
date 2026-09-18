import Chunk, { ChunkInfo } from "./chunk";
import { s3LikeFinishUpload, s3LikeUploadCallback, s3LikeUploadChunk } from "../api";
import { Status } from "./base";
import { PolicyType } from "../../../../api/explorer.ts";

export default class KS3 extends Chunk {
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
    await s3LikeFinishUpload(this.task.session!.completeURL, false, this.task.chunkProgress, this.cancelToken.token);

    this.logger.info(`Sending S3-like upload callback...`);
    return s3LikeUploadCallback(this.task.session!.session_id, this.task.session!.callback_secret, PolicyType.ks3);
  }
}
