import * as React from "react";
import { useLayoutEffect, useRef, useState } from "react";
import { Box, ListItemIcon, ListItemText, Menu, MenuItem, Typography, useMediaQuery, useTheme } from "@mui/material";
import { StyledTab, StyledTabs } from "./StyledComponents.tsx";
import CaretDown from "../Icons/CaretDown.tsx";
import { useTranslation } from "react-i18next";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";

export interface Tab<T> {
  label: React.ReactNode;
  value: T;
  icon?: React.ReactElement;
}

export interface ResponsiveTabsProps<T> {
  tabs: Tab<T>[];
  value: T;
  onChange: (event: React.SyntheticEvent, value: T) => void;
}

const ResponsiveTabs = <T,>({ tabs, value, onChange }: ResponsiveTabsProps<T>) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [hideTabs, setHideTabs] = useState(false);
  const tabsRef = useRef<HTMLDivElement>(null);
  const { t } = useTranslation();
  const moreOptionState = usePopupState({
    variant: "popover",
    popupId: "tabMore",
  });
  const { onClose, ...menuProps } = bindMenu(moreOptionState);
  useLayoutEffect(() => {
    const checkOverflow = () => {
      if (tabsRef.current?.children[0]?.children[0]) {
        setHideTabs((e) =>
          e
            ? true
            : (tabsRef.current?.children[0]?.children[0]?.scrollWidth ?? 0) >
              (tabsRef.current?.children[0]?.children[0]?.clientWidth ?? 0),
        );
      }
    };

    checkOverflow();
    window.addEventListener("resize", checkOverflow);
    return () => window.removeEventListener("resize", checkOverflow);
  }, []);

  return (
    <Box
      sx={{
        borderBottom: 1,
        borderColor: "divider",
        pb: "2px",
      }}
    >
      <StyledTabs ref={tabsRef} value={value} onChange={onChange}>
        {tabs
          .filter((tab) => (isMobile || hideTabs ? tab.value == value : true))
          .map((tab) => (
            <StyledTab label={tab.label} value={tab.value} icon={tab.icon} />
          ))}
        {(isMobile || hideTabs) && tabs.length > 1 && (
          <>
            <StyledTab
              label={
                <Typography
                  sx={{
                    display: "flex",
                    gap: "4px",
                  }}
                  variant={"inherit"}
                >
                  {t("application:navbar.showMore")}
                  <CaretDown sx={{ fontSize: 15 }} />
                </Typography>
              }
              {...bindTrigger(moreOptionState)}
            />
            <Menu {...menuProps} onClose={onClose}>
              {tabs
                .filter((tab) => tab.value != value)
                .map((option, index) => (
                  <MenuItem
                    dense
                    key={index}
                    onClick={(e) => {
                      onClose();
                      onChange(e, option.value);
                    }}
                  >
                    {option.icon && <ListItemIcon>{option.icon}</ListItemIcon>}
                    <ListItemText>{option.label}</ListItemText>
                  </MenuItem>
                ))}
            </Menu>
          </>
        )}
      </StyledTabs>
    </Box>
  );
};

export default ResponsiveTabs;
