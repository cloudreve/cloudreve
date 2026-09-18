import type React from "react";

/**
 * 从 Touch 事件中获取两个触控中心位置
 */
export default function getMultipleTouchPosition(
  evt: TouchEvent | React.TouchEvent,
): [clientX: number, clientY: number, touchLength: number] {
  const { clientX, clientY } = evt.touches[0];
  if (evt.touches.length >= 2) {
    const { clientX: nextClientX, clientY: nextClientY } = evt.touches[1];
    return [
      (clientX + nextClientX) / 2,
      (clientY + nextClientY) / 2,
      Math.sqrt((nextClientX - clientX) ** 2 + (nextClientY - clientY) ** 2),
    ];
  }
  return [clientX, clientY, 0];
}
