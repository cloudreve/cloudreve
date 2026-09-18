import { StoragePolicy, UploadCredential } from "../../../api/explorer.ts";
import { ChunkProgress } from "./uploader/chunk";

export enum TaskType {
  file,
  resumeHint,
}

export interface Task {
  type: TaskType;
  name: string;
  size: number;
  policy: StoragePolicy;
  dst: string;
  file: File;
  blob: Blob;
  child?: Task[];
  session?: UploadCredential;
  chunkProgress: ChunkProgress[];
  resumed: boolean;
  overwrite?: boolean;
}

type Nullable<T> = T | null;

export interface OneDriveError {
  error: {
    code: string;
    message: string;
    innererror?: {
      code: string;
    };
    retryAfterSeconds?: number;
  };
}

export interface OneDriveChunkResponse {
  expirationDateTime: string;
  nextExpectedRanges: string[];
}

export interface QiniuChunkResponse {
  etag: string;
  md5: string;
}

export interface QiniuError {
  error: string;
}

export interface QiniuPartsInfo {
  etag: string;
  partNumber: number;
}

export interface QiniuFinishUploadRequest {
  parts: QiniuPartsInfo[];
  mimeType?: string;
}

export interface UpyunError {
  message: string;
  code: number;
}

export interface S3Part {
  ETag: string;
  PartNumber: number;
}
