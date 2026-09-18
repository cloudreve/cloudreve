import { useReducer, useRef } from "react";
import type { ActiveAnimationType } from "../types";
import useForkedVariable from "./useForkedVariable";

/**
 * 动画关闭处理真实关闭状态
 * 通过 onAnimationEnd 回调实现 leaveCallback
 */
export default function useAnimationVisible(
  visible: boolean | undefined,
  afterClose?: () => void,
): [realVisible: boolean | undefined, activeAnimation: ActiveAnimationType, onAnimationEnd: () => void] {
  const [, handleRender] = useReducer((c) => !c, false);

  const activeAnimation = useRef<ActiveAnimationType>(0);

  // 可见状态分支
  const [realVisible, modifyRealVisible] = useForkedVariable(visible, (modify) => {
    // 可见状态：设置进入动画
    if (visible) {
      modify(visible);
      activeAnimation.current = 1;
    } else {
      activeAnimation.current = 2;
    }
  });

  function onAnimationEnd() {
    // 动画结束后触发渲染
    handleRender();
    // 结束动画：设置隐藏状态
    if (activeAnimation.current === 2) {
      modifyRealVisible(false);
      // 触发隐藏回调
      if (afterClose) {
        afterClose();
      }
    }
    // 重置状态
    activeAnimation.current = 0;
  }

  return [
    /**
     * 真实可见状态
     */
    realVisible,
    /**
     * 正在进行的动画
     */
    activeAnimation.current,
    /**
     * 动画结束后回调
     */
    onAnimationEnd,
  ];
}
