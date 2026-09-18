/**
 * 是否支持触摸设备
 */
const isTouchDevice = typeof window !== "undefined" && "ontouchstart" in window;

export default isTouchDevice;
