import { useRef } from "react";

export default function useInitial<T extends (...args: any) => any>(callback: T) {
  const { current } = useRef({ sign: false, fn: undefined as ReturnType<T> });
  if (!current.sign) {
    current.sign = true;
    current.fn = callback();
  }
  return current.fn;
}
