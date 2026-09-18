import React from "react";
import i18next from "i18next";
import { languages } from "../../i18n";
import { useTranslation } from "react-i18next";
import { IconButton, Menu, MenuItem, Tooltip } from "@mui/material";
import Translate from "../Icons/Translate.tsx";

const LanguageSwitcher: React.FC = () => {
  const { t } = useTranslation();
  const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  return (
    <>
      <Tooltip title={t("login.switchLanguage")}>
        <IconButton onClick={handleClick} sx={{ ml: 1 }}>
          <Translate />
        </IconButton>
      </Tooltip>
      <Menu anchorEl={anchorEl} open={open} onClose={handleClose}>
        {languages.map((lang) => (
          <MenuItem
            key={lang.code}
            selected={i18next.language === lang.code}
            onClick={() => {
              i18next.changeLanguage(lang.code);
              handleClose();
            }}
            sx={{ fontSize: 14 }}
          >
            {lang.displayName}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

export default LanguageSwitcher;
