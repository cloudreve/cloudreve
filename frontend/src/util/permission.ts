import { FileResponse } from "../api/explorer.ts";
import CrUri, { Filesystem } from "./uri.ts";

// canCopyMoveTo checks if the files can be copied or moved to the destination.
export function canCopyMoveTo(files: FileResponse[], dst: string, isCopy: boolean): boolean {
  const dstUri = new CrUri(dst);
  const srcUri = new CrUri(files[0].path);
  if (isCopy) {
    return srcUri.fs() == dstUri.fs() && srcUri.fs() == Filesystem.my;
  } else {
    switch (srcUri.fs()) {
      case Filesystem.my:
        return dstUri.fs() == Filesystem.my || dstUri.fs() == Filesystem.trash;
      case Filesystem.trash:
        return dstUri.fs() == Filesystem.my;
    }
  }

  return false;
}
