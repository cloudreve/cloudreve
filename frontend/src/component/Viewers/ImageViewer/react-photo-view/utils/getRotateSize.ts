/**
 * 获取旋转后的宽高
 */
export default function getRotateSize(rotate: number, width: number, height: number) {
  const isVertical = rotate % 180 !== 0;

  // 若图片不是水平则调换属性
  if (isVertical) {
    return [height, width, isVertical] as const;
  }

  return [width, height, isVertical] as const;
}
