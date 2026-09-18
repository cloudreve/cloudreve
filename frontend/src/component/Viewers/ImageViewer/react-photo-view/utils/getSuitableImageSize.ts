import { longModeRatio } from "../variables";
import getRotateSize from "./getRotateSize";

/**
 * 获取图片合适的大小
 */
export default function getSuitableImageSize(
  naturalWidth: number,
  naturalHeight: number,
  rotate: number,
  dynamicInnerWidth: number,
) {
  const [currentWidth, currentHeight, isVertical] = getRotateSize(rotate, dynamicInnerWidth, innerHeight);

  let y = 0;
  let width = currentWidth;
  let height = currentHeight;

  // 自适应宽高
  const autoWidth = (naturalWidth / naturalHeight) * currentHeight;
  const autoHeight = (naturalHeight / naturalWidth) * currentWidth;

  if (naturalWidth < currentWidth && naturalHeight < currentHeight) {
    width = naturalWidth;
    height = naturalHeight;
  } else if (naturalWidth < currentWidth && naturalHeight >= currentHeight) {
    width = autoWidth;
  } else if (naturalWidth >= currentWidth && naturalHeight < currentHeight) {
    height = autoHeight;
  } else if (naturalWidth / naturalHeight > currentWidth / currentHeight) {
    height = autoHeight;
  }
  // 长图模式
  else if (naturalHeight / naturalWidth >= longModeRatio && !isVertical) {
    height = autoHeight;
    y = (height - currentHeight) / 2;
  } else {
    width = autoWidth;
  }
  return {
    width,
    height,
    x: 0,
    y,
    pause: true,
  };
}
