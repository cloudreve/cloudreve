import React from "react";
import { createPortal } from "react-dom";
import "./SlidePortal.less";

export interface ISliderPortalProps extends React.HTMLAttributes<HTMLDivElement> {
  container?: HTMLElement;
}

function SlidePortal({ container = document.body, ...rest }: ISliderPortalProps) {
  return createPortal(<div {...rest} />, container);
}

export default SlidePortal;
