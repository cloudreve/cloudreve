import { Box, Button, Divider, Popover, styled, Tooltip, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { CSSProperties, useCallback, useState } from "react";
import { bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { bindPopover } from "material-ui-popup-state";
import Sketch from "@uiw/react-color-sketch";

export interface CircleColorSelectorProps {
  colors: string[];
  selectedColor: string;
  onChange: (color: string) => void;
  showColorValueInCustomization?: boolean;
}

export const customizeMagicColor = "-";

export const SelectorBox = styled(Box)({
  display: "flex",
  flexWrap: "wrap",
  gap: 4,
});

interface ColorCircleProps {
  color: string;
  selected: boolean;
  isCustomization?: boolean;
  onClick: (e: React.MouseEvent) => void;
  size?: number;
  noMb?: boolean;
}

const ColorCircleBox = styled("div")(({
  color,
  selected,
  size = 20,

  noMb,
}: {
  color: string;
  selected: boolean;
  size?: number;
  noMb?: boolean;
}) => {
  return {
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    width: `${size}px`,
    height: `${size}px`,
    padding: "3px",
    borderRadius: "50%",
    marginRight: 0,
    marginTop: 0,
    marginBottom: noMb ? 0 : "4px",
    boxSizing: "border-box",
    transform: "scale(1)",
    boxShadow: `${color} 0px 0px ${selected ? 5 : 0}px`,
    transition: "transform 100ms ease 0s, box-shadow 100ms ease 0s",
    background: color,
    ":hover": {
      transform: "scale(1.2)",
    },
  };
});

const ColorCircleBoxChild = styled("div")(({ selected }: { selected: boolean }) => {
  const theme = useTheme();
  return {
    "--circle-point-background-color": theme.palette.background.default,
    height: selected ? "100%" : 0,
    width: selected ? "100%" : 0,
    borderRadius: "50%",
    backgroundColor: "var(--circle-point-background-color)",
    boxSizing: "border-box",
    transition: "height 100ms ease 0s, width 100ms ease 0s",
    transform: "scale(0.5)",
  };
});

export const ColorCircle = ({ color, selected, isCustomization, onClick, size, noMb }: ColorCircleProps) => {
  const { t } = useTranslation();
  const displayColor = isCustomization
    ? "conic-gradient(red, yellow, lime, aqua, blue, magenta, red)"
    : color == ""
      ? "linear-gradient(45deg, rgba(217,217,217,1) 46%, rgba(217,217,217,1) 47%, rgba(128,128,128,1) 47%)"
      : color;
  return (
    <Tooltip title={isCustomization ? t("application:fileManager.customizeColor") : ""}>
      <ColorCircleBox size={size} onClick={onClick} color={displayColor} selected={selected} noMb={noMb}>
        <ColorCircleBoxChild selected={selected} />
      </ColorCircleBox>
    </Tooltip>
  );
};

const CircleColorSelector = (props: CircleColorSelectorProps) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const [customizeColor, setCustomizeColor] = useState<string>(props.selectedColor);
  const popupState = usePopupState({
    variant: "popover",
    popupId: "color-picker",
  });

  const onClick = useCallback(
    (color: string) => () => {
      if (color === customizeMagicColor) {
        return;
      }
      props.onChange(color);
    },
    [props.onChange],
  );

  const { onClose, ...restPopover } = bindPopover(popupState);
  const onApply = () => {
    onClose();
    onClick(customizeColor)();
  };
  return (
    <SelectorBox>
      {props.colors.map((color) => (
        <ColorCircle
          noMb={props.showColorValueInCustomization}
          isCustomization={color === customizeMagicColor && !props.showColorValueInCustomization}
          color={!props.showColorValueInCustomization ? color : props.selectedColor}
          onClick={onClick(color)}
          selected={color === props.selectedColor}
          {...(color === customizeMagicColor && bindTrigger(popupState))}
        />
      ))}
      <Popover
        {...restPopover}
        anchorOrigin={{
          vertical: "bottom",
          horizontal: "center",
        }}
        transformOrigin={{
          vertical: "top",
          horizontal: "center",
        }}
        onClose={onClose}
      >
        <Sketch
          presetColors={false}
          style={
            {
              border: "none",
              boxShadow: "none",
              padding: 0,
              margin: 0,
              background: theme.palette.background.default + "!important",
            } as CSSProperties
          }
          disableAlpha={true}
          color={customizeColor}
          onChange={(color) => {
            setCustomizeColor(color.hex);
          }}
        />
        <Divider />
        <Box sx={{ p: 1 }}>
          <Button size={"small"} onClick={onApply} fullWidth variant={"contained"}>
            {t("application:fileManager.apply")}
          </Button>
        </Box>
      </Popover>
    </SelectorBox>
  );
};

export default CircleColorSelector;
