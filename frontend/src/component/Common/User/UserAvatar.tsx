import { Avatar, Skeleton } from "@mui/material";
import { grey } from "@mui/material/colors";
import { bindHover, bindPopover } from "material-ui-popup-state";
import { usePopupState } from "material-ui-popup-state/hooks";
import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useInView } from "react-intersection-observer";
import { ApiPrefix } from "../../../api/request.ts";
import { User } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { loadUserInfo } from "../../../redux/thunks/session.ts";
import InPrivate from "../../Icons/InPrivate.tsx";
import UserPopover from "./UserPopover.tsx";

export interface UserAvatarProps {
  user?: User;
  uid?: string;
  overwriteTextSize?: boolean;
  onUserLoaded?: (user: User) => void;
  enablePopover?: boolean;
  square?: boolean;
  cacheKey?: string;
  [key: string]: any;
}

function stringToColor(string: string) {
  let hash = 0;
  let i;

  /* eslint-disable no-bitwise */
  for (i = 0; i < string.length; i += 1) {
    hash = string.charCodeAt(i) + ((hash << 5) - hash);
  }

  let color = "#";

  for (i = 0; i < 3; i += 1) {
    const value = (hash >> (i * 8)) & 0xff;
    color += `00${value.toString(16)}`.slice(-2);
  }
  /* eslint-enable no-bitwise */

  return color;
}

export const AnonymousUser: User = {
  id: "",
  nickname: "",
  created_at: "",
};

const UserAvatar = memo(
  ({
    user,
    key,
    overwriteTextSize,
    onUserLoaded,
    uid,
    sx,
    square,
    cacheKey,
    enablePopover,
    ...rest
  }: UserAvatarProps) => {
    const [loadedUser, setLoadedUser] = useState<User | undefined>(undefined);
    const dispatch = useAppDispatch();
    const popupState = usePopupState({
      variant: "popover",
      popupId: "user",
    });
    const { t } = useTranslation();
    const { ref, inView } = useInView({
      triggerOnce: true,
      rootMargin: "200px 0px",
      skip: !!user,
    });
    useEffect(() => {
      if (inView && !loadedUser) {
        if (uid) {
          loadUser(uid);
        }
      }
    }, [inView]);
    useEffect(() => {
      if (user) {
        setLoadedUser(user);
      }
    }, [user]);
    const loadUser = useCallback(
      async (uid: string) => {
        try {
          const u = await dispatch(loadUserInfo(uid));
          setLoadedUser(u);
          if (onUserLoaded) {
            onUserLoaded(u);
          }
        } catch (e) {
          console.warn("Failed to load user info", e);
        }
      },
      [dispatch, setLoadedUser, onUserLoaded],
    );

    const avatarUrl = useMemo(() => {
      if (loadedUser) {
        return ApiPrefix + `/user/avatar/${loadedUser.id}${cacheKey ? `?nocache=1&key=${cacheKey ?? 0}` : ""}`;
      }
      return undefined;
    }, [loadedUser, cacheKey]);

    return (
      <>
        {loadedUser && (
          <>
            <Avatar
              alt={loadedUser.nickname ?? t("modals.anonymous")}
              src={avatarUrl}
              slotProps={{
                img: {
                  loading: "lazy",
                  alt: "",
                },
              }}
              {...rest}
              {...bindHover(popupState)}
              sx={[
                {
                  bgcolor: loadedUser.nickname ? stringToColor(loadedUser.nickname) : grey[500],
                  ...sx,
                },
                overwriteTextSize && {
                  fontSize: `${sx.width * 0.6}px!important`,
                },
                square && {
                  borderRadius: (theme) => `${theme.shape.borderRadius}px`,
                },
              ]}
            >
              {loadedUser.id == "" ? (
                <InPrivate
                  sx={{
                    ...(overwriteTextSize
                      ? {
                          fontSize: `${sx.width * 0.6}px`,
                        }
                      : {}),
                  }}
                />
              ) : undefined}
            </Avatar>
            {enablePopover && user && <UserPopover user={user} {...bindPopover(popupState)} />}
          </>
        )}
        {!loadedUser && <Skeleton ref={ref} variant={"circular"} sx={{ ...sx }} {...rest} />}
      </>
    );
  },
);

export default UserAvatar;
