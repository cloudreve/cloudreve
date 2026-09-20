/**
 * Android Motion Photo (a.k.a. MicroVideo) support.
 *
 * Google/Samsung motion photos are a single JPEG with an MP4 appended at the
 * end. XMP metadata in the JPEG header either carries `GCamera:MicroVideoOffset`
 * (bytes from EOF to the start of the MP4) or a `Container:Directory` whose
 * `video/mp4` item declares `Item:Length` (video is the last item).
 */

// XMP packets live in the JPEG APP1 segment near the start of the file;
// 128 KiB of head data covers every layout seen in the wild.
const HEAD_SCAN_BYTES = 128 * 1024;

// Motion photos carry a multi-MB video tail; smaller JPEGs cannot be one.
const MIN_FILE_SIZE = 512 * 1024;

const MOTION_PHOTO_EXTENSIONS = new Set(["jpg", "jpeg"]);

export const isMotionPhotoCandidate = (name: string, size: number): boolean =>
  MOTION_PHOTO_EXTENSIONS.has(name.toLowerCase().split(".").pop() ?? "") &&
  size >= MIN_FILE_SIZE;

const bytesToLatin1 = (buf: ArrayBuffer): string => {
  const bytes = new Uint8Array(buf);
  let out = "";
  for (let i = 0; i < bytes.length; i += 8192) {
    out += String.fromCharCode.apply(
      null,
      bytes.subarray(i, i + 8192) as unknown as number[],
    );
  }
  return out;
};

// Extracts the embedded video's byte length (counted from EOF) from XMP text.
const extractVideoLength = (text: string): number | null => {
  const offset = text.match(/MicroVideoOffset="(\d+)"/);
  if (offset) {
    return parseInt(offset[1], 10);
  }

  if (!/MicroVideo|MotionPhoto/.test(text)) {
    return null;
  }

  const mimeIdx = text.indexOf("video/mp4");
  if (mimeIdx < 0) {
    return null;
  }
  // The item element serializes Mime before Length; search forward only so a
  // preceding image item's Length is never picked up.
  const window = text.slice(mimeIdx, mimeIdx + 512);
  const length = window.match(/Item:Length="(\d+)"/);
  return length ? parseInt(length[1], 10) : null;
};

// MP4 files start with an `ftyp` box; guards against bogus XMP offsets.
const looksLikeMp4 = (buf: ArrayBuffer): boolean => {
  if (buf.byteLength < 12) {
    return false;
  }
  const view = new DataView(buf, 4);
  return view.getUint32(0) === 0x66747970; // "ftyp"
};

/**
 * Returns a blob URL for the embedded MP4 of a motion photo, or null when the
 * image has no embedded video or the bytes cannot be fetched (CORS, network).
 * Uses HTTP Range requests so at most the head + video tail are transferred.
 */
export const fetchMotionPhotoVideoUrl = async (
  imageUrl: string,
): Promise<string | null> => {
  try {
    const head = await fetch(imageUrl, {
      headers: { Range: `bytes=0-${HEAD_SCAN_BYTES - 1}` },
    });
    if (!head.ok) {
      return null;
    }
    const headBuf = await head.arrayBuffer();

    const videoLength = extractVideoLength(bytesToLatin1(headBuf));
    if (!videoLength || videoLength <= 0) {
      return null;
    }

    let videoBuf: ArrayBuffer;
    if (head.status === 206 && headBuf.byteLength <= HEAD_SCAN_BYTES) {
      const tail = await fetch(imageUrl, {
        headers: { Range: `bytes=-${videoLength}` },
      });
      if (!tail.ok) {
        return null;
      }
      videoBuf = await tail.arrayBuffer();
    } else {
      // Range ignored: the head fetch already holds the whole file.
      if (headBuf.byteLength <= videoLength) {
        return null;
      }
      videoBuf = headBuf.slice(headBuf.byteLength - videoLength);
    }

    if (!looksLikeMp4(videoBuf)) {
      return null;
    }
    return URL.createObjectURL(new Blob([videoBuf], { type: "video/mp4" }));
  } catch {
    return null;
  }
};
