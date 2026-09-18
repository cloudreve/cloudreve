import type React from "react";
import { FileResponse } from "../../../../api/explorer.ts";

/**
 * 资源数据类型
 */
export interface DataType {
  /**
   * 唯一标识
   */
  key: number | string;
  /**
   * 资源地址
   */
  src?: string;
  /**
   * 自定义渲染，优先级比 src 低
   */
  render?: (props: PhotoRenderParams) => React.ReactNode;
  /**
   * 自定义覆盖节点
   */
  overlay?: React.ReactNode;
  /**
   * 指定渲染节点宽度
   */
  width?: number;
  /**
   * 指定渲染节点高度
   */
  height?: number;
  /**
   * 触发 ref
   */
  originRef?: React.MutableRefObject<HTMLElement | null>;

  // Cloudreve specific
  file?: FileResponse;
  version?: string;

  loadMorePlaceholder?: boolean;
}

export interface PhotoProviderBase {
  /**
   * 是否循环预览，达到该数量则启用
   * @defaultValue 3
   */
  loop?: boolean | number;
  /**
   * 动画速度
   * @defaultValue 400
   */
  speed?: (type: ActiveAnimationType) => number;
  /**
   * 动画函数
   * @defaultValue 'cubic-bezier(0.25, 0.8, 0.25, 1)'
   */
  easing?: (type: ActiveAnimationType) => string;
  /**
   * 图片点击是否可关闭
   */
  photoClosable?: boolean;
  /**
   * 背景点击是否可关闭
   * @defaultValue true
   */
  maskClosable?: boolean;
  /**
   * 默认背景透明度
   * 设置 null 背景不响应下拉变化
   * @defaultValue 1
   */
  maskOpacity?: number | null;
  /**
   * 下拉是否可关闭
   * @defaultValue true
   */
  pullClosable?: boolean;
  /**
   * 导航条 visible
   * @defaultValue true
   */
  bannerVisible?: boolean;
  /**
   * 自定义渲染覆盖物
   */
  overlayRender?: (overlayProps: OverlayRenderProps) => React.ReactNode;
  /**
   * 自定义渲染工具栏
   */
  toolbarRender?: (overlayProps: OverlayRenderProps) => React.ReactNode;
  className?: string;
  maskClassName?: string;
  photoWrapClassName?: string;
  photoClassName?: string;
  /**
   * 自定义 loading
   */
  loadingElement?: JSX.Element;
  /**
   * 自定义加载失败渲染
   */
  brokenElement?: JSX.Element | ((photoProps: BrokenElementParams) => JSX.Element);
  /**
   * @defaultValue document.body
   */
  portalContainer?: HTMLElement;
}

export type PhotoRenderParams = {
  /**
   * 自定义渲染 DOM 属性
   */
  attrs: Partial<React.HTMLAttributes<HTMLElement>>;
  scale: number;
  rotate: number;
};

/**
 * brokenElement 函数参数
 */
export interface BrokenElementParams {
  src: string;
}

export interface OverlayRenderProps {
  /**
   * 图片列表
   */
  images: DataType[];
  /**
   * 当前索引
   */
  index: number;
  /**
   * 索引改变回调
   */
  onIndexChange: (index: number) => void;
  /**
   * 是否可见
   */
  visible: boolean;
  /**
   * 关闭事件回调
   */
  onClose: (evt?: React.MouseEvent | React.TouchEvent) => void;
  /**
   * 覆盖物是否可见
   */
  overlayVisible: boolean;
  /**
   * 自定义覆盖节点
   */
  overlay?: React.ReactNode;
  /**
   * 当前旋转角度
   */
  rotate: number;
  /**
   * 旋转事件回调
   */
  onRotate: (rotate: number) => void;
  /**
   * 当前缩放
   */
  scale: number;
  /**
   * 缩放事件回调
   */
  onScale: (scale: number) => void;
}

export interface ExposedProperties {
  // 缩放
  scale?: number;
  // 旋转
  rotate?: number;
  // 缩放回调
  onScale?: (scale: number) => void;
  // 旋转回调
  onRotate?: (rotate: number) => void;
}

export type ReachMoveFunction = (reachPosition: ReachType, clientX: number, clientY: number, scale?: number) => void;

export type ReachFunction = (clientX: number, clientY: number) => void;

export type PhotoTapFunction = (clientX: number, clientY: number) => void;

/**
 * 边缘超出状态
 */
export type CloseEdgeType =
  | 1 // 小于屏幕宽度
  | 2 // 抵触左边/上边
  | 3 // 抵触右边/下边
  | undefined; // 正常滑动

/**
 * 边缘触发状态
 */
export type ReachType =
  | "x" // x 轴
  | "y" // y 轴
  | undefined; // 未触发

/**
 * 初始响应状态
 */
export type TouchStartType =
  | 0 // 未触发
  | 1 // X 轴优先
  | 2 // Y 轴往上 push
  | 3; // Y 轴往下 pull

export type OriginRectType = {
  // top
  T: number;
  // left
  L: number;
  // width
  W: number;
  // height
  H: number;
  // object-fit
  FIT: "contain" | "cover" | "fill" | undefined;
};

/**
 * 动画状态
 */
export type EasingMode =
  // 未初始化
  | 0
  // 进入：开始
  | 1
  // 进入：动画开始
  | 2
  // 进入：动画第二帧
  | 3
  // 正常
  | 4
  // 关闭
  | 5;

/**
 * 进行中的动画
 */
export type ActiveAnimationType =
  // 未初始化
  | 0
  // 进入
  | 1
  // 离开
  | 2
  // 切换
  | 3;
