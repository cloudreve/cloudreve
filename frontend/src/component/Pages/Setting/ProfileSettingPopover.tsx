import {
  Divider,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Popover,
  PopoverProps,
  SvgIconProps,
} from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ShareLinksInProfileLevel } from "../../../api/user.ts";
import Eye from "../../Icons/Eye.tsx";
import EyeOff from "../../Icons/EyeOff.tsx";
import Globe from "../../Icons/Globe.tsx";

export interface ProfileSettingPopoverProps extends PopoverProps {
  currentValue: ShareLinksInProfileLevel;
  onValueChange?: (value: ShareLinksInProfileLevel) => void;
  readOnly?: boolean;
}

const profileSettingOptions: {
  value: ShareLinksInProfileLevel;
  label: string;
  description: string;
  icon?: ((props: SvgIconProps) => JSX.Element) | typeof Eye;
}[] = [
  {
    value: ShareLinksInProfileLevel.public_share_only,
    label: "application:setting.publicShareOnly",
    description: "application:setting.publicShareOnlyDes",
    icon: Eye,
  },
  {
    value: ShareLinksInProfileLevel.all_share,
    label: "application:setting.allShare",
    description: "application:setting.allShareDes",
    icon: Globe,
  },
  {
    value: ShareLinksInProfileLevel.hide_share,
    label: "application:setting.hideShare",
    description: "application:setting.hideShareDes",
    icon: EyeOff,
  },
];

export const useProfileSettingSummary = (value: ShareLinksInProfileLevel) => {
  const { t } = useTranslation();

  const summary = useMemo(() => {
    const option = profileSettingOptions.find((opt) => opt.value === (value ?? ""));
    return option ? t(option.label) : t("application:setting.publicShareOnly");
  }, [value, t]);

  return summary;
};

const ProfileSettingPopover = ({ currentValue, onValueChange, readOnly, ...rest }: ProfileSettingPopoverProps) => {
  const { t } = useTranslation();

  const handleChange = (value: ShareLinksInProfileLevel) => () => {
    if (onValueChange && !readOnly) {
      onValueChange(value);
    }
  };

  return (
    <Popover
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "left",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "left",
      }}
      {...rest}
    >
      <List dense sx={{ maxWidth: 360 }} disablePadding>
        {profileSettingOptions.map((option, index) => (
          <>
            <ListItem disablePadding key={option.value}>
              <ListItemButton
                selected={currentValue === option.value}
                disabled={readOnly}
                onClick={handleChange(option.value)}
                sx={{ borderRadius: "0px" }}
                role={undefined}
                dense
              >
                {option.icon && (
                  <ListItemIcon sx={{ minWidth: "32px", margin: "0px 8px" }}>
                    <option.icon
                      sx={{
                        color: (theme) => (currentValue === option.value ? theme.palette.primary.main : "inherit"),
                      }}
                    />
                  </ListItemIcon>
                )}
                <ListItemText
                  primary={t(option.label)}
                  secondary={t(option.description)}
                  slotProps={{
                    secondary: {
                      variant: "body2",
                      sx: {
                        wordWrap: "break-word",
                      },
                    },
                  }}
                />
              </ListItemButton>
            </ListItem>
            {index < profileSettingOptions.length - 1 && <Divider />}
          </>
        ))}
        <Divider />
      </List>
    </Popover>
  );
};

export default ProfileSettingPopover;
