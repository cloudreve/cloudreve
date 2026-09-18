import { computePositionEdge } from "../utils/edgeHandle";
import getPositionOnMoveOrScale from "../utils/getPositionOnMoveOrScale";
import getRotateSize from "../utils/getRotateSize";
import { defaultSpeed, maxTouchTime } from "../variables";
import useMethods from "./useMethods";

// 触边运动反馈
const rebound = (start: number, bound: number, callback: (spatial: number) => boolean) =>
  easeOutMove(
    start,
    bound,
    callback,
    defaultSpeed / 4,
    (t) => t,
    () => easeOutMove(bound, start, callback),
  );

/**
 * 物理滚动到具体位置
 */
export default function useScrollPosition<C extends (spatial: number) => boolean>(
  callbackX: C,
  callbackY: C,
  callbackS: C,
  dynamicInnerWidth: number,
) {
  const callback = useMethods({
    X: (spatial: number) => callbackX(spatial),
    Y: (spatial: number) => callbackY(spatial),
    S: (spatial: number) => callbackS(spatial),
  });

  return (
    x: number,
    y: number,
    lastX: number,
    lastY: number,
    width: number,
    height: number,
    scale: number,
    safeScale: number,
    lastScale: number,
    rotate: number,
    touchedTime: number,
  ) => {
    const [currentWidth, currentHeight] = getRotateSize(rotate, width, height);
    // 开始状态下边缘触发状态
    const [beginEdgeX, beginX] = computePositionEdge(x, safeScale, currentWidth, dynamicInnerWidth);
    const [beginEdgeY, beginY] = computePositionEdge(y, safeScale, currentHeight, innerHeight);
    const moveTime = Date.now() - touchedTime;

    // 时间过长、超出安全范围的情况下不执行滚动逻辑，恢复安全范围
    if (moveTime >= maxTouchTime || safeScale !== scale || Math.abs(lastScale - scale) > 1) {
      // 计算中心缩放点
      const { x: nextX, y: nextY } = getPositionOnMoveOrScale(x, y, width, height, scale, safeScale);
      const targetX = beginEdgeX ? beginX : nextX !== x ? nextX : null;
      const targetY = beginEdgeY ? beginY : nextY !== y ? nextY : null;

      if (targetX !== null) {
        easeOutMove(x, targetX, callback.X);
      }
      if (targetY !== null) {
        easeOutMove(y, targetY, callback.Y);
      }
      if (safeScale !== scale) {
        easeOutMove(scale, safeScale, callback.S);
      }
      return;
    }

    // 初始速度
    const speedX = (x - lastX) / moveTime;
    const speedY = (y - lastY) / moveTime;
    const speedT = Math.sqrt(speedX ** 2 + speedY ** 2);
    // 是否接触到边缘
    let edgeX = false;
    let edgeY = false;

    scrollMove(speedT, (spatial) => {
      const nextX = x + spatial * (speedX / speedT);
      const nextY = y + spatial * (speedY / speedT);

      const [isEdgeX, currentX] = computePositionEdge(nextX, scale, currentWidth, innerWidth);
      const [isEdgeY, currentY] = computePositionEdge(nextY, scale, currentHeight, innerHeight);

      if (isEdgeX && !edgeX) {
        edgeX = true;
        if (beginEdgeX) {
          easeOutMove(nextX, currentX, callback.X);
        } else {
          rebound(currentX, nextX + (nextX - currentX), callback.X);
        }
      }

      if (isEdgeY && !edgeY) {
        edgeY = true;
        if (beginEdgeY) {
          easeOutMove(nextY, currentY, callback.Y);
        } else {
          rebound(currentY, nextY + (nextY - currentY), callback.Y);
        }
      }
      // 同时接触边缘的情况下停止滚动
      if (edgeX && edgeY) {
        return false;
      }

      const resultX = edgeX || callback.X(currentX);
      const resultY = edgeY || callback.Y(currentY);
      return resultX && resultY;
    });
  };
}

// 加速度
const acceleration = -0.001;
// 阻力
const resistance = 0.0002;

/**
 * 通过速度滚动到停止
 */
function scrollMove(initialSpeed: number, callback: (spatial: number) => boolean) {
  let v = initialSpeed;
  let s = 0;
  let lastTime: number | undefined;
  let frameId = 0;

  const calcMove = (now: number) => {
    if (!lastTime) {
      lastTime = now;
    }
    const dt = now - lastTime;
    const direction = Math.sign(initialSpeed);
    const a = direction * acceleration;
    const f = Math.sign(-v) * v ** 2 * resistance;
    const ds = v * dt + ((a + f) * dt ** 2) / 2;
    v += (a + f) * dt;

    s += ds;
    // move to s
    lastTime = now;

    if (direction * v <= 0) {
      caf();
      return;
    }

    if (callback(s)) {
      raf();
      return;
    }
    caf();
  };
  raf();

  function raf() {
    frameId = requestAnimationFrame(calcMove);
  }
  function caf() {
    cancelAnimationFrame(frameId);
  }
}

/**
 * 缓动函数
 */
const easeOutQuart = (x: number) => 1 - (1 - x) ** 4;

/**
 * 缓动回调
 */
function easeOutMove(
  start: number,
  end: number,
  callback: (spatial: number) => boolean,
  speed = defaultSpeed,
  easing = easeOutQuart,
  complete?: () => void,
) {
  const distance = end - start;
  if (distance === 0) {
    return;
  }

  const startTime = Date.now();
  let frameId = 0;

  const calcMove = () => {
    const time = Math.min(1, (Date.now() - startTime) / speed);
    const result = callback(start + easing(time) * distance);

    if (result && time < 1) {
      raf();
      return;
    }
    cancelAnimationFrame(frameId);
    if (time >= 1 && complete) {
      complete();
    }
  };
  raf();

  function raf() {
    frameId = requestAnimationFrame(calcMove);
  }
}
