import UserAvatar, { UserAvatarProps } from "./UserAvatar.tsx";
import { Skeleton, TypographyProps } from "@mui/material";
import { useState } from "react";
import { BadgeText, DefaultButton } from "../StyledComponents.tsx";
import { usePopupState } from "material-ui-popup-state/hooks";
import UserPopover from "./UserPopover.tsx";
import { bindHover, bindPopover } from "material-ui-popup-state";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

export interface UserBadgeProps extends UserAvatarProps {
  textProps?: TypographyProps;
}

const UserBadge = ({ textProps, user, uid, ...rest }: UserBadgeProps) => {
  const { t } = useTranslation();
  const [userLoaded, setUserLoaded] = useState(user);
  const navigate = useNavigate();
  const popupState = usePopupState({
    variant: "popover",
    popupId: "user",
  });
  return (
    <>
      <DefaultButton
        {...bindHover(popupState)}
        onClick={() => navigate(`/profile/${user?.id ?? uid}`)}
        sx={{
          display: "flex",
          alignItems: "center",
          maxWidth: "150px",
        }}
      >
        <UserAvatar overwriteTextSize user={user} onUserLoaded={(u) => setUserLoaded(u)} uid={uid} {...rest} />
        <BadgeText {...textProps}>
          {userLoaded ? (
            userLoaded.id ? (
              userLoaded.nickname
            ) : (
              t("application:modals.anonymous")
            )
          ) : (
            <Skeleton width={60} />
          )}
        </BadgeText>
      </DefaultButton>
      {userLoaded && <UserPopover user={userLoaded} {...bindPopover(popupState)} />}
    </>
  );
};

export default UserBadge;
