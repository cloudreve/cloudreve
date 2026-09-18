import CrUri from "../../../../util/uri";
import { UploaderError, UploaderErrorName } from "../errors";
import Logger from "../logger";
import { Task } from "../types";
import { ChunkProgress } from "../uploader/chunk";

// 文件分块
export function getChunks(file: Blob, chunkByteSize: number | undefined): Blob[] {
  // 如果 chunkByteSize 比文件大或为0，则直接取文件的大小
  if (!chunkByteSize || chunkByteSize > file.size || chunkByteSize == 0) {
    chunkByteSize = file.size;
  }

  const chunks: Blob[] = [];
  const count = Math.ceil(file.size / chunkByteSize);
  for (let i = 0; i < count; i++) {
    const chunk = file.slice(chunkByteSize * i, i === count - 1 ? file.size : chunkByteSize * (i + 1));
    chunks.push(chunk);
  }

  if (chunks.length == 0) {
    chunks.push(file.slice(0));
  }
  return chunks;
}

export function sumChunk(list: ChunkProgress[]) {
  return list.reduce((data, loaded) => data + loaded.loaded, 0);
}

const resumeKeyPrefix = "cd_upload_ctx_";

function isTask(toBeDetermined: Task | string): toBeDetermined is Task {
  return !!(toBeDetermined as Task).name;
}

export function getResumeCtxKey(task: Task | string): string {
  if (isTask(task)) {
    return `${resumeKeyPrefix}name_${task.name}_dst_${task.dst}_size_${task.size}_policy_${task.policy.id}`;
  }

  return task;
}

export function setResumeCtx(task: Task, logger: Logger) {
  const ctxKey = getResumeCtxKey(task);
  try {
    localStorage.setItem(ctxKey, JSON.stringify(task));
  } catch (err) {
    logger.warn(new UploaderError(UploaderErrorName.WriteCtxFailed, `setResumeCtx failed: ${ctxKey}`));
  }
}

export function removeResumeCtx(task: Task | string, logger: Logger) {
  const ctxKey = getResumeCtxKey(task);
  try {
    localStorage.removeItem(ctxKey);
  } catch (err) {
    logger.warn(new UploaderError(UploaderErrorName.RemoveCtxFailed, `removeResumeCtx failed. key: ${ctxKey}`));
  }
}

export function cleanupResumeCtx(logger: Logger) {
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key && key.startsWith(resumeKeyPrefix)) {
      try {
        localStorage.removeItem(key);
      } catch (err) {
        logger.warn(new UploaderError(UploaderErrorName.RemoveCtxFailed, `removeResumeCtx failed. key: ${key}`));
      }
    }
  }
}

export function getResumeCtx(task: Task | string, logger: Logger): Task | null {
  const ctxKey = getResumeCtxKey(task);
  let localInfoString: string | null = null;
  try {
    localInfoString = localStorage.getItem(ctxKey);
  } catch {
    logger.warn(new UploaderError(UploaderErrorName.ReadCtxFailed, `getResumeCtx failed. key: ${ctxKey}`));
  }

  if (localInfoString == null) {
    return null;
  }

  let localInfo: Task | null = null;
  try {
    localInfo = JSON.parse(localInfoString);
  } catch {
    // 本地信息已被破坏，直接删除
    removeResumeCtx(task, logger);
    logger.warn(new UploaderError(UploaderErrorName.InvalidCtxData, `getResumeCtx failed to parse. key: ${ctxKey}`));
  }

  if (localInfo && localInfo.session && localInfo.session.expires < Math.floor(Date.now() / 1000)) {
    removeResumeCtx(task, logger);
    logger.warn(
      new UploaderError(
        UploaderErrorName.CtxExpired,
        `upload session already expired at ${localInfo.session.expires}. key: ${ctxKey}`,
      ),
    );
    return null;
  }

  return localInfo;
}

export function listResumeCtx(logger: Logger): Task[] {
  const res: Task[] = [];
  for (let i = 0, len = localStorage.length; i < len; i++) {
    const key = localStorage.key(i);
    if (key && key.startsWith(resumeKeyPrefix)) {
      const value = getResumeCtx(key, logger);
      if (value) {
        res.push(value);
      }
    }
  }

  return res;
}

export function OBJtoXML(obj: any): string {
  let xml = "";
  for (const prop in obj) {
    xml += "<" + prop + ">";
    if (Array.isArray(obj[prop])) {
      for (const array of obj[prop]) {
        // A real botch fix here
        xml += "</" + prop + ">";
        xml += "<" + prop + ">";

        xml += OBJtoXML(new Object(array));
      }
    } else if (typeof obj[prop] == "object") {
      xml += OBJtoXML(new Object(obj[prop]));
    } else {
      xml += obj[prop];
    }
    xml += "</" + prop + ">";
  }
  return xml.replace(/<\/?[0-9]{1,}>/g, "");
}

export function getFileInput(id: number, isFolder: boolean): HTMLInputElement {
  const input = document.createElement("input");
  input.type = "file";
  input.id = `upload-file-input-${id}`;
  if (isFolder) {
    input.id = `upload-folder-input-${id}`;
    input.setAttribute("webkitdirectory", "true");
    input.setAttribute("mozdirectory", "true");
  } else {
    input.id = `upload-file-input-${id}`;
    input.multiple = true;
  }
  input.hidden = true;
  document.body.appendChild(input);
  return input;
}

export function pathJoin(parts: string[], sep = "/"): string {
  parts = parts.map((part, index) => {
    if (index) {
      part = part.replace(new RegExp("^" + sep), "");
    }
    if (index !== parts.length - 1) {
      part = part.replace(new RegExp(sep + "$"), "");
    }
    return part;
  });
  return parts.join(sep);
}

function basename(path: string): string {
  const pathList = path.split("/");
  pathList.pop();
  return pathList.join("/") === "" ? "/" : pathList.join("/");
}

export function trimPrefix(src: string, prefix: string): string {
  if (src.startsWith(prefix)) {
    return src.slice(prefix.length);
  }
  return src;
}

export function getDirectoryUploadDst(dst: string, file: any): string {
  let relPath = file.webkitRelativePath;
  if (!relPath || relPath == "") {
    relPath = file.fsPath;
    if (!relPath || relPath == "") {
      return dst;
    }
  }

  const dstCrUrl = new CrUri(dst);
  relPath = trimPrefix(relPath, "/");
  return dstCrUrl
    .join(...relPath.split("/"))
    .parent()
    .toString();
}

// Wrap readEntries in a promise to make working with readEntries easier
async function readEntriesPromise(directoryReader: any): Promise<any> {
  try {
    return await new Promise((resolve, reject) => {
      directoryReader.readEntries(resolve, reject);
    });
  } catch (err) {
    console.log(err);
  }
}

async function readFilePromise(fileReader: any, path: string): Promise<any> {
  try {
    return await new Promise((resolve, _reject) => {
      fileReader.file((file: any) => {
        file.fsPath = path;
        resolve(file);
      });
    });
  } catch (err) {
    console.log(err);
  }
}

// Get all the entries (files or sub-directories) in a directory by calling readEntries until it returns empty array
async function readAllDirectoryEntries(directoryReader: any): Promise<any> {
  const entries: any[] = [];
  let readEntries = await readEntriesPromise(directoryReader);
  while (readEntries.length > 0) {
    entries.push(...readEntries);
    readEntries = await readEntriesPromise(directoryReader);
  }
  return entries;
}

// Drop handler function to get all files
export async function getAllFileEntries(dataTransferItemList: DataTransferItemList): Promise<File[]> {
  const fileEntries: any[] = [];
  // Use BFS to traverse entire directory/file structure
  const queue: any[] = [];
  // Unfortunately dataTransferItemList is not iterable i.e. no forEach
  for (let i = 0; i < dataTransferItemList.length; i++) {
    const fileEntry = dataTransferItemList[i].webkitGetAsEntry();
    if (!fileEntry) {
      const file = dataTransferItemList[i].getAsFile();
      if (file) {
        fileEntries.push(file);
      }
    }

    queue.push(dataTransferItemList[i].webkitGetAsEntry());
  }
  while (queue.length > 0) {
    const entry = queue.shift();
    if (!entry) {
      continue;
    }
    if (entry.isFile) {
      fileEntries.push(await readFilePromise(entry, entry.fullPath));
    } else if (entry.isDirectory) {
      const reader = entry.createReader();
      const entries: any = await readAllDirectoryEntries(reader);
      queue.push(...entries);
    }
  }
  return fileEntries;
}

export function isFileDrop(e: DragEvent): boolean {
  return !!e.dataTransfer && e.dataTransfer.types.includes("Files");
}
