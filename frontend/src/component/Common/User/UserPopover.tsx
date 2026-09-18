import { Box, Button, PopoverProps, styled, Tooltip, Typography } from "@mui/material";
import HoverPopover from "material-ui-popup-state/HoverPopover";
import { useCallback, useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { User } from "../../../api/user.ts";
import UserAvatar from "./UserAvatar.tsx";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { loadUserInfo } from "../../../redux/thunks/session.ts";
import MailOutlined from "../../Icons/MailOutlined.tsx";
import HomeOutlined from "../../Icons/HomeOutlined.tsx";
import { useNavigate } from "react-router-dom";
import TimeBadge from "../TimeBadge.tsx";

interface UserPopoverProps extends PopoverProps {
  user: User;
}

const ActionButton = styled(Button)({
  minWidth: "initial",
});

export const UserProfile = ({ user, open, displayOnly }: { user: User; open: boolean; displayOnly?: boolean }) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [loadedUser, setLoadedUser] = useState(user);
  useEffect(() => {
    if (open && user.id && !user.group) {
      dispatch(loadUserInfo(user.id)).then((u) => {
        if (u) {
          setLoadedUser(u);
        }
      });
    }
  }, [open]);

  return (
    <Box
      sx={{
        display: "flex",
      }}
    >
      <UserAvatar overwriteTextSize user={user} sx={{ width: 80, height: 80 }} />
      <Box sx={{ ml: 2 }}>
        <Box sx={{ display: "flex", alignItems: "center" }}>
          <Typography variant={"h6"} fontWeight={600}>
            {user.id ? user.nickname : t("application:modals.anonymous")}
          </Typography>
          {displayOnly && (
            <Typography variant={"body2"} sx={{ ml: 1 }} color={"text.secondary"}>
              {loadedUser?.group ? loadedUser.group.name : ""}
            </Typography>
          )}
        </Box>
        {!displayOnly && (
          <Typography variant={"body2"} color={"text.secondary"}>
            {loadedUser?.group ? loadedUser.group.name : ""}
          </Typography>
        )}
        {displayOnly && loadedUser?.created_at && (
          <Typography variant={"body2"} color={"text.secondary"}>
            <Trans
              i18nKey={"setting.accountCreatedAt"}
              ns={"application"}
              components={[<TimeBadge variant={"inherit"} datetime={loadedUser.created_at} />]}
            />
          </Typography>
        )}
        <Box
          sx={{
            mt: 0.5,
            position: "relative",
            left: -10,
            display: "flex",
          }}
        >
          {!displayOnly && user.id && (
            <Tooltip title={t("application:setting.profilePage")}>
              <ActionButton onClick={() => navigate(`/profile/${user.id}`)}>
                <HomeOutlined />
              </ActionButton>
            </Tooltip>
          )}
          {user.email && (
            <Tooltip title={user.email}>
              <ActionButton onClick={() => window.open(`mailto:${user.email}`)}>
                <MailOutlined />
              </ActionButton>
            </Tooltip>
          )}
        </Box>
      </Box>
    </Box>
  );
};

const UserPopover = ({ user, open, ...rest }: UserPopoverProps) => {
  const stopPropagation = useCallback((e: any) => e.stopPropagation(), []);
  return (
    <HoverPopover
      onMouseDown={stopPropagation}
      onMouseUp={stopPropagation}
      onClick={stopPropagation}
      open={open}
      {...rest}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "left",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "left",
      }}
    >
      <Box sx={{ minWidth: "300px", p: 2 }}>
        <UserProfile user={user} open={open} />
      </Box>
    </HoverPopover>
  );
};

export default UserPopover;
