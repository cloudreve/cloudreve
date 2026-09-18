import { upyunFormUploadChunk } from "../api";
import Base from "./base";
import { EncryptedBlob } from "./encrypt/blob";

export default class Upyun extends Base {
  public upload = async () => {
    this.logger.info("Starting uploading file stream:", this.task.file);
    if (this.task.blob instanceof EncryptedBlob && !this.task.policy.streaming_encryption) {
      this.task.blob = new Blob([await this.task.blob.bytes()]);
    }
    await upyunFormUploadChunk(
      this.task.session?.upload_urls[0]!,
      this.task.blob,
      this.task.session?.upload_policy!,
      this.task.session?.credential!,
      (p) => {
        this.subscriber.onProgress({
          total: this.getProgressInfoItem(p.loaded, p.total ?? 1),
        });
      },
      this.cancelToken.token,
      this.task.session?.mime_type,
    );
  };
}
