import { IconButton, Popover, ToggleButton, ToggleButtonGroup, Tooltip } from "@mui/material";
import DarkTheme from "../../Icons/DarkTheme.tsx";
import { useTranslation } from "react-i18next";
import { useMemo, useState } from "react";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import Sunny from "../../Icons/Sunny.tsx";
import Moon from "../../Icons/Moon.tsx";
import SunWithTime from "../../Icons/SunWithTime.tsx";
import { setDarkMode } from "../../../redux/globalStateSlice.ts";
import SessionManager, { UserSettings } from "../../../session";

interface SwitchPopoverProps {
  open?: boolean;
  anchorEl?: HTMLElement | null;
  onClose?: () => void;
}

export const SwitchPopover = ({ open, anchorEl, onClose }: SwitchPopoverProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const darkMode = useAppSelector((state) => state.globalState.darkMode);
  const currentMode = useMemo(() => {
    if (darkMode === undefined) {
      return "system";
    }
    return darkMode ? "dark" : "light";
  }, [darkMode]);
  const handleChange = (_event: React.MouseEvent<HTMLElement>, newMode: string) => {
    let newSetting: boolean | undefined;
    if (newMode === "light") {
      newSetting = false;
    } else if (newMode === "dark") {
      newSetting = true;
    }
    dispatch(setDarkMode(newSetting));
    SessionManager.set(UserSettings.PreferredDarkMode, newSetting);
    onClose && onClose();
  };

  const inner = (
    <ToggleButtonGroup
      color="primary"
      value={currentMode}
      exclusive
      onChange={handleChange}
      size={onClose ? undefined : "small"}
      aria-label="Platform"
    >
      <ToggleButton value="light">
        <Sunny fontSize="small" sx={{ mr: 1 }} />
        {t("navbar.toLightMode")}
      </ToggleButton>
      <ToggleButton value="system">
        <SunWithTime fontSize="small" sx={{ mr: 1 }} />
        {t("setting.syncWithSystem")}
      </ToggleButton>
      <ToggleButton value="dark">
        <Moon fontSize="small" sx={{ mr: 1 }} />
        {t("navbar.toDarkMode")}
      </ToggleButton>
    </ToggleButtonGroup>
  );

  return onClose ? (
    <Popover
      open={!!open}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "left",
      }}
    >
      {inner}
    </Popover>
  ) : (
    inner
  );
};

const DarkThemeSwitcher = () => {
  const { t } = useTranslation();
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  return (
    <>
      <Tooltip title={t("navbar.darkModeSwitch")}>
        <IconButton onClick={handleClick} size="large">
          <DarkTheme />
        </IconButton>
      </Tooltip>
      <SwitchPopover open={Boolean(anchorEl)} anchorEl={anchorEl} onClose={() => setAnchorEl(null)} />
    </>
  );
};

export default DarkThemeSwitcher;
