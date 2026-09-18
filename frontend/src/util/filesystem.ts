// get the paths of files (no directories) in the directory
// parent: "" or "/"
export const getFileSystemDirectoryPaths = async (
  handle: FileSystemDirectoryHandle,
  parent = "",
): Promise<Map<string, string>> => {
  const paths: Map<string, string> = new Map();

  for await (const [path, fileSystemHandle] of handle.entries()) {
    if (fileSystemHandle instanceof window.FileSystemFileHandle) {
      paths.set(`${parent}${path}`, "1");
    } else {
      (await getFileSystemDirectoryPaths(fileSystemHandle, `${parent}${path}/`)).forEach((value, key) => {
        paths.set(key, value);
      });
    }
  }

  return paths;
};

// create the dst directory if it doesn't exist
// return the dst directory handle
// paths: "/dir1/dir2" => ["dir1","dir2"]
export const createFileSystemDirectory = async (handle: FileSystemDirectoryHandle, paths: string[]) => {
  let cur = handle;
  while (paths.length > 0) {
    const path = paths.shift();
    if (!path) {
      break;
    }
    cur = await cur.getDirectoryHandle(path, { create: true });
  }
  return cur;
};

// save file into the dst directory
// create the dst file if it doesn't exist by default
// path: a/b/c.jpg
export const saveFileToFileSystemDirectory = async (
  handle: FileSystemDirectoryHandle,
  stream: FileSystemWriteChunkType,
  path: string,
  create = true,
) => {
  const paths = path.split("/");
  const fileName = paths.pop();
  if (!fileName) return;

  const dir = await createFileSystemDirectory(handle, paths);
  const file = await dir.getFileHandle(fileName, { create });
  const writable = await file.createWritable();
  await writable.write(stream);
  await writable.close();
};

// stream response body directly to a file in the dst directory,
// calling onProgress for each chunk written
export const streamFileToFileSystemDirectory = async (
  handle: FileSystemDirectoryHandle,
  reader: ReadableStreamDefaultReader<Uint8Array>,
  path: string,
  onProgress: (bytesWritten: number) => void,
  create = true,
) => {
  const paths = path.split("/");
  const fileName = paths.pop();
  if (!fileName) return;

  const dir = await createFileSystemDirectory(handle, paths);
  const file = await dir.getFileHandle(fileName, { create });
  const writable = await file.createWritable();
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      await writable.write(value as unknown as FileSystemWriteChunkType);
      onProgress(value.byteLength);
    }
    await writable.close();
  } catch (e) {
    await writable.abort();
    throw e;
  }
};

// verify or request the permission of the readwrite permission
export async function verifyFileSystemRWPermission(fileHandle: FileSystemDirectoryHandle) {
  const opts = { mode: "readwrite" as FileSystemPermissionMode };

  // Check if we already have permission, if so, return true.
  if ((await fileHandle.queryPermission(opts)) === "granted") {
    return true;
  }

  // Request permission to the file, if the user grants permission, return true.
  if ((await fileHandle.requestPermission(opts)) === "granted") {
    return true;
  }

  // The user did not grant permission, return false.
  return false;
}
