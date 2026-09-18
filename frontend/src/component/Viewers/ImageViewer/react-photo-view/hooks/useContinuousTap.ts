import { useRef } from "react";
import useDebounceCallback from "./useDebounceCallback";

export type TapFuncType<T> = (...args: T[]) => void;

/**
 * 单击和双击事件处理
 * @param singleTap - 单击事件
 * @param doubleTap - 双击事件
 * @return invokeTap
 */
export default function useContinuousTap<T>(singleTap: TapFuncType<T>, doubleTap: TapFuncType<T>): TapFuncType<T> {
  // 当前连续点击次数
  const continuousClick = useRef(0);

  const debounceTap = useDebounceCallback(
    (...args) => {
      continuousClick.current = 0;
      singleTap(...args);
    },
    {
      wait: 300,
    },
  );

  return function invokeTap(...args) {
    continuousClick.current += 1;
    debounceTap(...args);
    // 双击
    if (continuousClick.current >= 2) {
      debounceTap.cancel();
      continuousClick.current = 0;
      doubleTap(...args);
    }
  };
}
