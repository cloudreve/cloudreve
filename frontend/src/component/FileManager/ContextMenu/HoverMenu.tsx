import * as React from "react";
import { Menu, type MenuProps } from "@mui/material";

const HoverMenu: React.ComponentType<MenuProps> = React.forwardRef(function HoverMenu(props: MenuProps, ref): any {
  return (
    <Menu
      {...props}
      ref={ref}
      style={{ pointerEvents: "none", ...props.style }}
      slotProps={{
        ...props.slotProps,
        paper: {
          ...props.slotProps?.paper,
          style: {
            pointerEvents: "auto",
          },
        },
      }}
    />
  );
});

export default HoverMenu;
