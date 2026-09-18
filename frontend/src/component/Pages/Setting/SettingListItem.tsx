import {
  Avatar,
  ListItem,
  ListItemAvatar,
  ListItemProps,
  ListItemSecondaryAction,
  styled,
  SvgIconProps,
} from "@mui/material";
import { StyledListItemText } from "../../Common/StyledComponents.tsx";
import * as React from "react";

const StyledListItem = styled(ListItem)(({ theme }) => ({
  borderRadius: theme.shape.borderRadius,
  border: `1px solid ${theme.palette.mode === "light" ? "rgba(0, 0, 0, 0.23)" : "rgba(255, 255, 255, 0.23)"}`,
  marginTop: theme.spacing(1),
  paddingBottom: theme.spacing(0.5),
  paddingTop: theme.spacing(0.5),
}));

export interface SettingListItemProps extends ListItemProps {
  icon: (props: SvgIconProps) => JSX.Element;
  iconColor?: string;
  settingTitle?: React.ReactNode;
  settingDescription?: React.ReactNode;
  settingAction?: React.ReactNode;
}

const SettingListItem = ({
  icon,
  iconColor,
  settingTitle,
  settingDescription,
  settingAction,
  ...rest
}: SettingListItemProps) => {
  const Icon = icon;
  return (
    <StyledListItem {...rest}>
      <ListItemAvatar sx={{}}>
        <Avatar sx={{ backgroundColor: iconColor }}>
          <Icon />
        </Avatar>
      </ListItemAvatar>
      <StyledListItemText
        primary={settingTitle}
        secondaryTypographyProps={{
          variant: "caption",
          sx: {
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          },
        }}
        secondary={settingDescription}
      />
      {settingAction && <ListItemSecondaryAction>{settingAction}</ListItemSecondaryAction>}
    </StyledListItem>
  );
};

export default SettingListItem;
