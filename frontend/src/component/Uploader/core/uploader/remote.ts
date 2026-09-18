import Chunk, { ChunkInfo } from "./chunk";
import { slaveUploadChunk } from "../api";

export default class Remote extends Chunk {
  protected async uploadChunk(chunkInfo: ChunkInfo) {
    return slaveUploadChunk(
      `${this.task.session?.upload_urls[0]!}`,
      this.task.session?.credential!,
      chunkInfo,
      (p) => {
        this.updateChunkProgress(p.loaded, chunkInfo.index);
      },
      this.cancelToken.token,
    );
  }
}
