import {
  Box,
  Divider,
  IconButton,
  ListItemIcon,
  ListItemText,
  MenuList,
  Popover,
  PopoverProps,
  styled,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { bindPopover } from "material-ui-popup-state";
import { bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { GroupPermission } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { signout } from "../../../redux/thunks/session.ts";
import SessionManager, { Session } from "../../../session";
import { GroupBS } from "../../../session/utils.ts";
import UserAvatar from "../../Common/User/UserAvatar.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import HomeOutlined from "../../Icons/HomeOutlined.tsx";
import Person from "../../Icons/Person.tsx";
import SettingsOutlined from "../../Icons/SettingsOutlined.tsx";
import SignOut from "../../Icons/SignOut.tsx";
import WrenchSettings from "../../Icons/WrenchSettings.tsx";

const StyledTypography = styled(Typography)(() => ({
  lineHeight: 1,
}));

const UserPopover = ({ open, onClose, ...rest }: PopoverProps) => {
  const user = SessionManager.currentUser();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  if (!user) {
    return null;
  }

  const isAdmin = useMemo(() => {
    return GroupBS(user).enabled(GroupPermission.is_admin);
  }, [user.group?.permission]);

  const signWithHint = (email: string) => {
    navigate("/session?phase=email&email=" + encodeURIComponent(email));
  };

  const signOut = useCallback(() => {
    dispatch(signout());
    onClose && onClose({}, "backdropClick");
  }, []);

  const openMyProfile = useCallback(() => {
    navigate(`/profile/${user?.id}`);
    onClose && onClose({}, "backdropClick");
  }, [user?.id]);

  const openSetting = useCallback(() => {
    navigate(`/settings`);
    onClose && onClose({}, "backdropClick");
  }, [user?.id]);

  const openDashboard = useCallback(() => {
    navigate(`/admin/home`);
    onClose && onClose({}, "backdropClick");
  }, [user?.id]);

  return (
    <Popover
      open={open}
      onClose={onClose}
      {...rest}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "right",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "right",
      }}
    >
      <Box
        sx={{
          px: "12px",
          py: "10px",
          minWidth: "230px",
          display: "flex",
          maxWidth: "300px",
          width: "100%",
          flexDirection: "column",
          gap: "4px",
        }}
      >
        <Box sx={{ display: "flex" }}>
          <StyledTypography paragraph={false} variant={"body2"} fontWeight={600}>
            {user.nickname}
          </StyledTypography>
          <StyledTypography variant={"body2"} paragraph={false} color={"textSecondary"} sx={{ ml: 1 }}>
            {user.group?.name}
          </StyledTypography>
        </Box>
        <StyledTypography variant={"caption"} color={"textSecondary"}>
          {user.email}
        </StyledTypography>
      </Box>
      <Divider />
      <MenuList dense sx={{ mx: 0.5 }}>
        {isAdmin && (
          <SquareMenuItem onClick={openDashboard}>
            <ListItemIcon>
              <WrenchSettings fontSize={"small"} />
            </ListItemIcon>
            <ListItemText>{t("navbar.dashboard")}</ListItemText>
          </SquareMenuItem>
        )}
        {isMobile && (
          <SquareMenuItem onClick={openSetting}>
            <ListItemIcon>
              <SettingsOutlined fontSize={"small"} />
            </ListItemIcon>
            <ListItemText>{t("navbar.setting")}</ListItemText>
          </SquareMenuItem>
        )}
        <SquareMenuItem onClick={openMyProfile}>
          <ListItemIcon>
            <HomeOutlined fontSize={"small"} />
          </ListItemIcon>
          <ListItemText>{t("navbar.myProfile")}</ListItemText>
        </SquareMenuItem>
        <SquareMenuItem onClick={signOut}>
          <ListItemIcon>
            <SignOut />
          </ListItemIcon>
          <ListItemText>{t("login.logout")}</ListItemText>
        </SquareMenuItem>
      </MenuList>
    </Popover>
  );
};

const UserAction = () => {
  const navigate = useNavigate();
  const [current, setCurrent] = useState<Session>();
  const popupState = usePopupState({ variant: "popover", popupId: "user" });
  useEffect(() => {
    try {
      const session = SessionManager.currentLogin();
      if (session) {
        setCurrent(session);
      }
    } catch (e) {}
  }, []);
  return (
    <>
      <IconButton size={current ? "large" : undefined} {...(current ? bindTrigger(popupState) : {})}>
        {!current && <Person onClick={() => navigate("/session")} />}
        {current && <UserAvatar sx={{ width: 30, height: 30 }} user={current.user} />}
      </IconButton>
      <UserPopover {...bindPopover(popupState)} />
    </>
  );
};

export default UserAction;
