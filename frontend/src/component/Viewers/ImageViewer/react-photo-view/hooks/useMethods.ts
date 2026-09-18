import { useRef } from "react";

/**
 * Hook of persistent methods
 */
export default function useMethods<T extends Record<string, (...args: any[]) => any>>(fn: T) {
  const { current } = useRef({
    fn,
    curr: undefined as T | undefined,
  });
  current.fn = fn;

  if (!current.curr) {
    const curr = Object.create(null);
    Object.keys(fn).forEach((key) => {
      curr[key] = (...args: unknown[]) => current.fn[key].call(current.fn, ...args);
    });
    current.curr = curr;
  }

  return current.curr as T;
}
