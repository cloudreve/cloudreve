import { ProcessingTaskDuplicatedError } from "../errors";
import Base from "../uploader/base";

export interface QueueContent {
  uploader: Base;
  resolve: () => void;
  reject: (err?: any) => void;
}

export class Pool {
  queue: Array<QueueContent> = [];
  processing: Array<QueueContent> = [];
  onPoolEmpty?: () => void;

  constructor(
    public limit: number,
    onPoolEmpty?: () => void,
  ) {
    this.onPoolEmpty = onPoolEmpty;
  }

  enqueue(uploader: Base) {
    return new Promise<void>((resolve, reject) => {
      this.queue.push({
        uploader,
        resolve,
        reject,
      });
      this.check();
    });
  }

  release(item: QueueContent) {
    this.processing = this.processing.filter((v) => v !== item);
    if (this.queue.length === 0 && this.processing.length === 0) {
      this.onPoolEmpty?.();
    }
    this.check();
  }

  run(item: QueueContent) {
    this.queue = this.queue.filter((v) => v !== item);
    if (
      this.processing.findIndex(
        (v) => v.uploader.task.dst == item.uploader.task.dst && v.uploader.task.file.name == item.uploader.task.name,
      ) > -1
    ) {
      // 找到重名任务
      item.reject(new ProcessingTaskDuplicatedError());
      this.release(item);
      return;
    }

    this.processing.push(item);
    item.uploader.run().then(
      () => {
        item.resolve();
        this.release(item);
      },
      (err) => {
        item.reject(err);
        this.release(item);
      },
    );
  }

  check() {
    const processingNum = this.processing.length;
    const availableNum = Math.max(0, this.limit - processingNum);
    this.queue.slice(0, availableNum).forEach((item) => {
      this.run(item);
    });
  }
}
