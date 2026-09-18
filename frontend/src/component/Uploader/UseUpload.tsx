import React, { useEffect, useRef, useState } from "react";
import Base, { RetryOption } from "./core/uploader/base.ts";
import { useSnackbar } from "notistack";

export function useUpload(uploader: Base) {
  const startLoadedRef = useRef(0);
  const [status, setStatus] = useState(uploader.status);
  const [error, setError] = useState(uploader.error);
  const [progress, setProgress] = useState(uploader.progress);
  const { enqueueSnackbar } = useSnackbar();

  useEffect(() => {
    /* eslint-disable @typescript-eslint/no-empty-function */
    uploader.subscribe({
      onTransition: (newStatus) => {
        setStatus(newStatus);
      },
      onError: (err) => {
        setError(err);
        setStatus(uploader.status);
      },
      onProgress: (data) => {
        setProgress(data);
      },
      onMsg: (msg, color) => {
        enqueueSnackbar(msg, { variant: color });
      },
    });
  }, []);

  // 获取上传速度
  const [speed, speedAvg] = React.useMemo(() => {
    if (progress == undefined || progress.total == null || progress.total.loaded == null) return [0, 0];
    const duration = (Date.now() - (uploader.lastTime || 0)) / 1000;
    const durationTotal = (Date.now() - (uploader.startTime || 0)) / 1000;
    const res =
      progress.total.loaded > startLoadedRef.current
        ? Math.floor((progress.total.loaded - startLoadedRef.current) / duration)
        : 0;
    const resAvg = progress.total.loaded > 0 ? Math.floor(progress.total.loaded / durationTotal) : 0;

    startLoadedRef.current = progress.total.loaded;
    uploader.lastTime = Date.now();
    return [res, resAvg];
  }, [progress?.total.loaded]);

  const retry = (opt?: RetryOption) => {
    uploader.retry(opt);
  };

  return { status, error, progress, speed, speedAvg, retry };
}
