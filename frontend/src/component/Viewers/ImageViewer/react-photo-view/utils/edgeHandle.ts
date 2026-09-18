import type { CloseEdgeType, ReachType, TouchStartType } from "../types";

/**
 * 获取接触边缘类型
 */
export const getReachType = (
  initialTouchState: TouchStartType,
  horizontalCloseEdge: CloseEdgeType,
  verticalCloseEdge: CloseEdgeType,
  reachPosition: ReachType,
): ReachType => {
  if ((horizontalCloseEdge && initialTouchState === 1) || reachPosition === "x") {
    return "x";
  }
  if ((verticalCloseEdge && initialTouchState > 1) || reachPosition === "y") {
    return "y";
  }
  return undefined;
};

/**
 * 计算接触边缘位置
 * @param position - x/y
 * @param scale
 * @param size - width/height
 * @param innerSize - innerWidth/innerHeight
 * @return [CloseEdgeType, position]
 */
export const computePositionEdge = (position: number, scale: number, size: number, innerSize: number) => {
  const currentWidth = size * scale;
  // 图片超出的宽度
  const outOffset = (currentWidth - innerSize) / 2;
  let closedEdge: CloseEdgeType;

  let current = position;
  if (currentWidth <= innerSize) {
    closedEdge = 1;
    current = 0;
  } else if (position > 0 && outOffset - position <= 0) {
    closedEdge = 2;
    current = outOffset;
  } else if (position < 0 && outOffset + position <= 0) {
    closedEdge = 3;
    current = -outOffset;
  }

  return [closedEdge, current] as const;
};
