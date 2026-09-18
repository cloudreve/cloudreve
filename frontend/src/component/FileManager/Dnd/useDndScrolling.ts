import { useRef } from "react";
import { throttle } from "lodash";

const threshold = 0.1;

const useDragScrolling = (containers: string[]) => {
  const isScrolling = useRef(false);
  const targets = containers.map((id) => document.querySelector(id) as HTMLElement);
  const rects = useRef<DOMRect[]>([]);

  const goDown = (target: HTMLElement) => {
    return () => {
      target.scrollTop += 5;

      const { offsetHeight, scrollTop, scrollHeight } = target;
      const isScrollEnd = offsetHeight + scrollTop >= scrollHeight;

      if (isScrolling.current && !isScrollEnd) {
        window.requestAnimationFrame(goDown(target));
      }
    };
  };

  const goUp = (target: HTMLElement) => {
    return () => {
      target.scrollTop -= 5;
      if (isScrolling.current && target.scrollTop > 0) {
        window.requestAnimationFrame(goUp(target));
      }
    };
  };

  const onDragOver = (event: MouseEvent) => {
    // detect if mouse is in any rect
    rects.current.forEach((rect, index) => {
      if (event.clientX < rect.left || event.clientX > rect.right) {
        isScrolling.current = false;
        return;
      }

      const height = rect.bottom - rect.top;
      if (event.clientY > rect.top && event.clientY < rect.top + threshold * height) {
        isScrolling.current = true;
        window.requestAnimationFrame(goUp(targets[index]));
      } else if (event.clientY < rect.bottom && event.clientY > rect.bottom - threshold * height) {
        isScrolling.current = true;
        window.requestAnimationFrame(goDown(targets[index]));
      } else {
        isScrolling.current = false;
      }
    });
  };

  const throttleOnDragOver = throttle(onDragOver, 300);

  const addEventListenerForWindow = () => {
    rects.current = targets.map((t) => t.getBoundingClientRect());
    window.addEventListener("dragover", throttleOnDragOver, false);
  };

  const removeEventListenerForWindow = () => {
    window.removeEventListener("dragover", throttleOnDragOver, false);
    isScrolling.current = false;
  };

  return {
    addEventListenerForWindow,
    removeEventListenerForWindow,
  };
};

export default useDragScrolling;
