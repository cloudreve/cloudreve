import { useRef, useMemo } from "react";

/**
 * 逻辑分叉变量处理
 * 此 hook 不触发额外渲染
 */
export default function useForkedVariable<T>(initial: T, updater: (modify: (variable: T) => void) => void) {
  // 初始分叉变量
  const forkedRef = useRef(initial);

  function modify(next: T) {
    forkedRef.current = next;
  }

  useMemo(() => {
    // 参数变化之后同步内部分叉变量
    updater(modify);
  }, [initial]);

  return [forkedRef.current, modify] as const;
}
