import Chunk, { ChunkInfo } from "./chunk";
import { qiniuDriveUploadChunk, qiniuFinishUpload } from "../api";
import { Status } from "./base";
import { Task } from "../types";
import UploadManager from "../index";
import * as utils from "../utils";

export default class ResumeHint extends Chunk {
  constructor(task: Task, manager: UploadManager) {
    super(task, manager);
    this.status = Status.resumable;
    this.progress = {
      total: this.getProgressInfoItem(utils.sumChunk(this.task.chunkProgress), this.task.size + 1),
    };
    this.subscriber.onProgress(this.progress);
  }

  protected async uploadChunk(chunkInfo: ChunkInfo) {
    return null;
  }
}
