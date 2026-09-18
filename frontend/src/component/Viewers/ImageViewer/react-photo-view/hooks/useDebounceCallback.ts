import { useCallback, useRef } from "react";

interface DebounceCallback<CallbackArguments extends any[]> {
  (...args: CallbackArguments): void;
  cancel: () => void;
}

export default function useDebounceCallback<CallbackArguments extends any[]>(
  callback: (...args: CallbackArguments) => void,
  {
    leading = false,
    maxWait,
    wait = maxWait || 0,
  }: {
    leading?: boolean;
    maxWait?: number;
    wait?: number;
  },
): DebounceCallback<CallbackArguments> {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  const prev = useRef(0);
  const trailingTimeout = useRef<ReturnType<typeof setTimeout>>();
  const clearTrailing = () => trailingTimeout.current && clearTimeout(trailingTimeout.current);

  const fn = useCallback(
    (...args: CallbackArguments) => {
      const now = Date.now();

      function call() {
        prev.current = now;
        clearTrailing();
        callbackRef.current.apply(null, args);
      }
      const last = prev.current;
      const offset = now - last;
      // leading
      if (last === 0) {
        if (leading) {
          call();
        }
        prev.current = now;
      }

      // body
      if (maxWait !== undefined) {
        if (offset > maxWait) {
          call();
          return;
        }
      } else if (offset < wait) {
        prev.current = now;
      }

      // trailing
      clearTrailing();
      trailingTimeout.current = setTimeout(() => {
        call();
        prev.current = 0;
      }, wait);
    },
    [wait, maxWait, leading],
  );
  (fn as DebounceCallback<CallbackArguments>).cancel = clearTrailing;

  return fn as DebounceCallback<CallbackArguments>;
}
