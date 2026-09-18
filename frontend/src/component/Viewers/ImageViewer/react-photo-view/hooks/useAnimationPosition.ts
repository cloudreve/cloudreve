import type { MutableRefObject } from "react";
import useAnimationOrigin from "./useAnimationOrigin";
import useTargetScale from "./useTargetScale";

export default function useAnimationPosition(
  dynamicInnerWidth: number,
  visible: boolean | undefined,
  originRef: MutableRefObject<HTMLElement | null> | undefined,
  loaded: boolean,
  x: number,
  y: number,
  width: number,
  height: number,
  scale: number,
  speed: number,
  updateEasing: (pause: boolean) => void,
) {
  // 延迟更新 width/height
  const [autoWidth, autoHeight, autoScale] = useTargetScale(width, height, scale, speed, updateEasing);
  // 动画源处理
  const [easingMode, originRect] = useAnimationOrigin(visible, originRef, loaded, speed, updateEasing);

  // 计算动画位置
  const { T, L, W, H, FIT } = originRect;
  // 偏移量，x: 0, y: 0 居中为初始
  const centerWidth = dynamicInnerWidth / 2;
  const centerHeight = innerHeight / 2;
  const offsetX = centerWidth - (width * scale) / 2;
  const offsetY = centerHeight - (height * scale) / 2;
  // 缩略图状态
  const miniMode = easingMode < 3 || easingMode > 4;
  // 有缩略图时，则为缩略图的位置，否则居中
  const translateX = miniMode ? (W ? L : centerWidth) : x + offsetX;
  const translateY = miniMode ? (W ? T : centerHeight) : y + offsetY;

  // 最小值缩放
  const minScale = W / (width * scale) || 0.01;

  // 适应 objectFit 保持缩略图宽高比
  const currentHeight = miniMode && FIT ? autoWidth * (H / W) : autoHeight;
  // 初始加载情况无缩放
  const currentScale = easingMode === 0 ? autoScale : miniMode ? minScale : autoScale;
  const opacity = miniMode ? (FIT ? 1 : 0) : 1;

  return [translateX, translateY, autoWidth, currentHeight, currentScale, opacity, easingMode, FIT] as const;
}
