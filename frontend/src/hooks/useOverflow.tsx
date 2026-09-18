import { useLayoutEffect, useState } from "react";

export const useIsOverflow = (ref: React.RefObject<HTMLElement>, callback?: (o: boolean) => void) => {
  const [isOverflow, setIsOverflow] = useState(false);

  useLayoutEffect(() => {
    const { current } = ref;
    if (current == null) {
      return;
    }

    const trigger = () => {
      const hasOverflow = current.scrollWidth > current.clientWidth;

      setIsOverflow(hasOverflow);

      if (callback) callback(hasOverflow);
    };

    if (current) {
      if ("ResizeObserver" in window) {
        const observer = new ResizeObserver(trigger);
        observer.observe(current);
        return () => observer.unobserve(current);
      }
    }
  }, [callback, ref]);

  return isOverflow;
};
