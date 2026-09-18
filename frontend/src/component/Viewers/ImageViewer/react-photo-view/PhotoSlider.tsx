import { Box, IconButton, Tooltip, useMediaQuery, useTheme } from "@mui/material";
import React, { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { setSidebar } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { downloadSingleFile } from "../../../../redux/thunks/download.ts";
import { openFileContextMenu } from "../../../../redux/thunks/file.ts";
import { switchToImageEditor } from "../../../../redux/thunks/viewer.ts";
import { fileExtension } from "../../../../util";
import useActionDisplayOpt, { canUpdate } from "../../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import { FileManagerIndex } from "../../../FileManager/FileManager.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Download from "../../../Icons/Download.tsx";
import ImageEdit from "../../../Icons/ImageEdit.tsx";
import Info from "../../../Icons/Info.tsx";
import MoreHorizontal from "../../../Icons/MoreHorizontal.tsx";
import { editorSupportedExt } from "../ImageEditor.tsx";
import ArrowLeft from "./components/ArrowLeft";
import ArrowRight from "./components/ArrowRight";
import SlidePortal from "./components/SlidePortal";
import useAdjacentImages from "./hooks/useAdjacentImages";
import useAnimationVisible from "./hooks/useAnimationVisible";
import useEventListener from "./hooks/useEventListener";
import useIsomorphicLayoutEffect from "./hooks/useIsomorphicLayoutEffect";
import useMethods from "./hooks/useMethods";
import useSetState from "./hooks/useSetState";
import PhotoBox from "./PhotoBox";
import "./PhotoSlider.less";
import type { DataType, OverlayRenderProps, PhotoProviderBase, ReachType } from "./types";
import isTouchDevice from "./utils/isTouchDevice";
import { limitNumber } from "./utils/limitTarget";
import { defaultEasing, defaultOpacity, defaultSpeed, horizontalOffset, maxMoveOffset } from "./variables";

export interface IPhotoSliderProps extends PhotoProviderBase {
  // 图片列表
  images: DataType[];
  // 图片当前索引
  index?: number;
  // 索引改变回调
  onIndexChange?: (index: number) => void;
  // 可见
  visible: boolean;
  // 关闭回调
  onClose: (evt?: React.MouseEvent | React.TouchEvent) => void;
  // 关闭动画结束后回调
  afterClose?: () => void;
  moreFiles?: boolean;
}

type PhotoSliderState = {
  // 偏移量
  x: number;
  // 图片处于触摸的状态
  touched: boolean;
  // 是否暂停 transition
  pause: boolean;
  // Reach 开始时 x 坐标
  lastCX: number | undefined;
  // Reach 开始时 y 坐标
  lastCY: number | undefined;
  // 背景透明度
  bg: number | null | undefined;
  // 上次关闭的背景透明度
  lastBg: number | null | undefined;
  // 是否显示 overlay
  overlay: boolean;
  // 是否为最小状态，可下拉关闭
  minimal: boolean;
  // 缩放
  scale: number;
  // 旋转
  rotate: number;
  // 缩放回调
  onScale?: (scale: number) => void;
  // 旋转回调
  onRotate?: (rotate: number) => void;
};

const initialState: PhotoSliderState = {
  x: 0,
  touched: false,
  pause: false,
  lastCX: undefined,
  lastCY: undefined,
  bg: undefined,
  lastBg: undefined,
  overlay: true,
  minimal: true,
  scale: 1,
  rotate: 0,
};

export default function PhotoSlider(props: IPhotoSliderProps) {
  const {
    loop = 3,
    speed: speedFn,
    easing: easingFn,
    photoClosable,
    maskClosable = true,
    maskOpacity = defaultOpacity,
    pullClosable = true,
    bannerVisible = true,
    overlayRender,
    toolbarRender,
    className,
    maskClassName,
    photoClassName,
    photoWrapClassName,
    loadingElement,
    brokenElement,
    images,
    index: controlledIndex = 0,
    onIndexChange: controlledIndexChange,
    visible,
    onClose,
    afterClose,
    portalContainer,
    moreFiles,
  } = props;

  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const containerRef = useRef(null);
  const sideBarOpen = useAppSelector((state) => state.globalState.sidebarOpen);
  const [state, updateState] = useSetState(initialState);
  const [innerIndex, updateInnerIndex] = useState(0);
  const dynamicInnerWidth = sideBarOpen ? window.innerWidth - 300 : window.innerWidth;

  const {
    x,
    touched,
    pause,

    lastCX,
    lastCY,

    bg = maskOpacity,
    lastBg,
    overlay,
    minimal,

    scale,
    rotate,
    onScale,
    onRotate,
  } = state;

  // 受控 index
  const isControlled = props.hasOwnProperty("index");
  const index = isControlled ? controlledIndex : innerIndex;
  const onIndexChange = isControlled ? controlledIndexChange : updateInnerIndex;
  // 内部虚拟 index
  const virtualIndexRef = useRef(index);

  // 当前图片
  const imageLength = images.length;
  const currentImage: DataType | undefined = images[index];

  // 是否开启
  // noinspection SuspiciousTypeOfGuard
  const enableLoop = typeof loop === "boolean" ? loop : imageLength > loop;

  // 显示动画处理
  const [realVisible, activeAnimation, onAnimationEnd] = useAnimationVisible(visible, afterClose);

  useEffect(() => {
    setTimeout(() => {
      document.body.style.overflow = "hidden";
    }, 500);
    return () => {
      setTimeout(() => {
        document.body.style.overflow = isMobile ? "initial" : "hidden";
      }, 500);
    };
  }, []);

  useIsomorphicLayoutEffect(() => {
    // 显示弹出层，修正正确的指向
    if (realVisible) {
      updateState({
        pause: true,
        x: index * -(dynamicInnerWidth + horizontalOffset),
      });
      virtualIndexRef.current = index;
      return;
    }
    // 关闭后清空状态
    updateState(initialState);
  }, [realVisible]);

  const { close, changeIndex } = useMethods({
    close(evt?: React.MouseEvent | React.TouchEvent) {
      if (onRotate) {
        onRotate(0);
      }
      updateState({
        overlay: true,
        // 记录当前关闭时的透明度
        lastBg: bg,
      });
      onClose(evt);
    },
    changeIndex(nextIndex: number, isPause: boolean = false) {
      // 当前索引
      const currentIndex = enableLoop ? virtualIndexRef.current + (nextIndex - index) : nextIndex;
      const max = imageLength - 1;
      // 虚拟 index
      // 非循环模式，限制区间
      const limitIndex = limitNumber(currentIndex, 0, max);
      const nextVirtualIndex = enableLoop ? currentIndex : limitIndex;
      // 单个屏幕宽度
      const singlePageWidth = dynamicInnerWidth + horizontalOffset;

      updateState({
        touched: false,
        lastCX: undefined,
        lastCY: undefined,
        x: -singlePageWidth * nextVirtualIndex,
        pause: isPause,
      });

      virtualIndexRef.current = nextVirtualIndex;
      // 更新真实的 index
      const realLoopIndex = nextIndex < 0 ? max : nextIndex > max ? 0 : nextIndex;
      if (onIndexChange) {
        onIndexChange(enableLoop ? realLoopIndex : limitIndex);
      }
    },
  });

  useEventListener("keydown", (evt: KeyboardEvent) => {
    if (visible) {
      switch (evt.key) {
        case "ArrowLeft":
          changeIndex(index - 1, true);
          break;
        case "ArrowRight":
          changeIndex(index + 1, true);
          break;
        case "Escape":
          close();
          break;
        default:
      }
    }
  });

  function handlePhotoTap(closeable: boolean | undefined) {
    return closeable ? close() : updateState({ overlay: !overlay });
  }

  const handleResize = (dynamicInnerWidth: number) => () => {
    updateState({
      x: -(dynamicInnerWidth + horizontalOffset) * index,
      lastCX: undefined,
      lastCY: undefined,
      pause: true,
    });
    virtualIndexRef.current = index;
  };

  function handleReachVerticalMove(clientY: number, nextScale?: number) {
    if (lastCY === undefined) {
      updateState({
        touched: true,
        lastCY: clientY,
        bg,
        minimal: true,
      });
      return;
    }
    const opacity =
      maskOpacity === null ? null : limitNumber(maskOpacity, 0.01, maskOpacity - Math.abs(clientY - lastCY) / 100 / 4);

    updateState({
      touched: true,
      lastCY,
      bg: nextScale === 1 ? opacity : maskOpacity,
      minimal: nextScale === 1,
    });
  }

  function handleReachHorizontalMove(clientX: number) {
    if (lastCX === undefined) {
      updateState({
        touched: true,
        lastCX: clientX,
        x,
        pause: false,
      });
      return;
    }
    const originOffsetClientX = clientX - lastCX;
    let offsetClientX = originOffsetClientX;

    // 第一张和最后一张超出距离减半
    if (
      !enableLoop &&
      ((index === 0 && originOffsetClientX > 0) || (index === imageLength - 1 && originOffsetClientX < 0))
    ) {
      offsetClientX = originOffsetClientX / 2;
    }

    updateState({
      touched: true,
      lastCX,
      x: -(dynamicInnerWidth + horizontalOffset) * virtualIndexRef.current + offsetClientX,
      pause: false,
    });
  }

  function handleReachMove(reachPosition: ReachType, clientX: number, clientY: number, nextScale?: number) {
    if (reachPosition === "x") {
      handleReachHorizontalMove(clientX);
    } else if (reachPosition === "y") {
      handleReachVerticalMove(clientY, nextScale);
    }
  }

  function handleReachUp(clientX: number, clientY: number) {
    const offsetClientX = clientX - (lastCX ?? clientX);
    const offsetClientY = clientY - (lastCY ?? clientY);
    let willClose = false;
    // 下一张
    if (offsetClientX < -maxMoveOffset) {
      changeIndex(index + 1);
      return;
    }
    // 上一张
    if (offsetClientX > maxMoveOffset) {
      changeIndex(index - 1);
      return;
    }
    const singlePageWidth = dynamicInnerWidth + horizontalOffset;
    // 当前偏移
    const currentTranslateX = -singlePageWidth * virtualIndexRef.current;

    if (Math.abs(offsetClientY) > 100 && minimal && pullClosable) {
      willClose = true;
      close();
    }
    updateState({
      touched: false,
      x: currentTranslateX,
      lastCX: undefined,
      lastCY: undefined,
      bg: maskOpacity,
      overlay: willClose ? true : overlay,
    });
  }
  // 截取相邻的图片
  const adjacentImages = useAdjacentImages(images, index, enableLoop);
  const currentFile = images[index]?.file;
  const displayOpt = useActionDisplayOpt(currentFile ? [currentFile] : []);

  useEffect(() => {
    //handleReachMove("x", 0, 0);
    handleReachUp(0, 0);
  }, [sideBarOpen]);

  if (!realVisible) {
    return null;
  }

  const currentOverlayVisible = overlay && !activeAnimation;
  // 关闭过程中使用下拉保存的透明度
  const currentOpacity = visible ? bg : lastBg;
  // 覆盖物参数
  const overlayParams: OverlayRenderProps | undefined = onScale &&
    onRotate && {
      images,
      index,
      visible,
      onClose: close,
      onIndexChange: changeIndex,
      overlayVisible: currentOverlayVisible,
      overlay: currentImage && currentImage.overlay,
      scale,
      rotate,
      onScale,
      onRotate,
    };
  // 动画时间
  const currentSpeed = speedFn ? speedFn(activeAnimation) : defaultSpeed;
  const currentEasing = easingFn ? easingFn(activeAnimation) : defaultEasing;
  const slideSpeed = speedFn ? speedFn(3) : defaultSpeed + 200;
  const slideEasing = easingFn ? easingFn(3) : defaultEasing;

  return (
    <Box ref={containerRef}>
      <SlidePortal
        className={`PhotoView-Portal${!currentOverlayVisible ? " PhotoView-Slider__clean" : ""}${
          !visible ? " PhotoView-Slider__willClose" : ""
        }${className ? ` ${className}` : ""}`}
        style={{
          width: sideBarOpen ? "calc(100% - 300px)" : "100%",
        }}
        role="dialog"
        onClick={(e) => e.stopPropagation()}
        container={portalContainer}
      >
        <div
          className={`PhotoView-Slider__Backdrop${maskClassName ? ` ${maskClassName}` : ""}${
            activeAnimation === 1
              ? " PhotoView-Slider__fadeIn"
              : activeAnimation === 2
                ? " PhotoView-Slider__fadeOut"
                : ""
          }`}
          style={{
            background: currentOpacity ? `rgba(0, 0, 0, ${currentOpacity})` : undefined,
            transitionTimingFunction: currentEasing,
            transitionDuration: `${touched ? 0 : currentSpeed}ms`,
            animationDuration: `${currentSpeed}ms`,
          }}
          onAnimationEnd={onAnimationEnd}
        />
        {bannerVisible && (
          <div className="PhotoView-Slider__BannerWrap">
            <div className="PhotoView-Slider__Counter">
              {index + 1} / {moreFiles ? imageLength - 1 : imageLength}
              {moreFiles ? "+" : ""}
            </div>
            <div className="PhotoView-Slider__BannerRight">
              {toolbarRender && overlayParams && toolbarRender(overlayParams)}
              {currentFile && displayOpt.showDownload && (
                <Tooltip title={t("application:fileManager.download")}>
                  <IconButton onClick={() => dispatch(downloadSingleFile(currentFile, images[index]?.version))}>
                    <Download fontSize={"small"} />
                  </IconButton>
                </Tooltip>
              )}
              {currentFile && displayOpt.showInfo && (
                <Tooltip title={t("application:fileManager.details")}>
                  <IconButton
                    onClick={() =>
                      dispatch(
                        setSidebar({
                          open: true,
                          target: images[index].file,
                        }),
                      )
                    }
                  >
                    <Info fontSize={"small"} />
                  </IconButton>
                </Tooltip>
              )}
              {currentFile &&
                displayOpt &&
                canUpdate(displayOpt) &&
                editorSupportedExt.includes(fileExtension(currentFile.name) ?? "") && (
                  <Tooltip title={t("application:fileManager.edit")}>
                    <IconButton onClick={() => dispatch(switchToImageEditor(currentFile, images[index]?.version))}>
                      <ImageEdit fontSize={"small"} />
                    </IconButton>
                  </Tooltip>
                )}
              {currentFile && (
                <Tooltip title={t("application:fileManager.moreActions")}>
                  <IconButton
                    onClick={(e) => dispatch(openFileContextMenu(FileManagerIndex.main, currentFile, false, e))}
                  >
                    <MoreHorizontal fontSize={"small"} />
                  </IconButton>
                </Tooltip>
              )}
              <IconButton onClick={close}>
                <Dismiss fontSize={"small"} />
              </IconButton>
            </div>
          </div>
        )}
        {adjacentImages.map((item: DataType, currentIndex) => {
          // 截取之前的索引位置
          const nextIndex =
            !enableLoop && index === 0 ? index + currentIndex : virtualIndexRef.current - 1 + currentIndex;

          return (
            <PhotoBox
              key={enableLoop ? `${item.key}/${item.src}/${nextIndex}` : item.key}
              item={item}
              speed={currentSpeed}
              easing={currentEasing}
              visible={visible}
              onReachMove={handleReachMove}
              onReachUp={handleReachUp}
              onPhotoTap={() => handlePhotoTap(photoClosable)}
              onMaskTap={() => handlePhotoTap(maskClosable)}
              wrapClassName={photoWrapClassName}
              className={photoClassName}
              style={{
                left: `${(dynamicInnerWidth + horizontalOffset) * nextIndex}px`,
                transform: `translate3d(${x}px, 0px, 0)`,
                transition: touched || pause ? undefined : `transform ${slideSpeed}ms ${slideEasing}`,
              }}
              loadingElement={loadingElement}
              brokenElement={brokenElement}
              onPhotoResize={handleResize(dynamicInnerWidth)}
              isActive={virtualIndexRef.current === nextIndex}
              expose={updateState}
            />
          );
        })}
        {!isTouchDevice && bannerVisible && (
          <>
            {(enableLoop || index !== 0) && (
              <div className="PhotoView-Slider__ArrowLeft" onClick={() => changeIndex(index - 1, true)}>
                <ArrowLeft />
              </div>
            )}
            {(enableLoop || index + 1 < imageLength) && (
              <div className="PhotoView-Slider__ArrowRight" onClick={() => changeIndex(index + 1, true)}>
                <ArrowRight />
              </div>
            )}
          </>
        )}
        {overlayRender && overlayParams && (
          <div className="PhotoView-Slider__Overlay">{overlayRender(overlayParams)}</div>
        )}
      </SlidePortal>
    </Box>
  );
}
