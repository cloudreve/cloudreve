import { LoadingButton } from "@mui/lab";
import { Collapse, Grid2, Stack, Typography, useMediaQuery, useTheme, styled } from "@mui/material";
import { bindPopover, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendUpdateUserSetting } from "../../../api/api.ts";
import { UserSettings, ShareLinksInProfileLevel } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import SessionManager from "../../../session";
import { DefaultButton, DenseFilledTextField } from "../../Common/StyledComponents.tsx";
import TimeBadge from "../../Common/TimeBadge.tsx";
import CaretDown from "../../Icons/CaretDown.tsx";
import AvatarSetting from "./AvatarSetting.tsx";
import ProfileSettingPopover, { useProfileSettingSummary } from "./ProfileSettingPopover.tsx";
import SettingForm from "./SettingForm.tsx";

export interface ProfileSettingProps {
  setting: UserSettings;
  setSetting: (setting: UserSettings) => void;
}

const ProfileDropButton = styled(DefaultButton)(({ theme }) => ({
  color: theme.palette.text.secondary,
  minWidth: 0,
  minHeight: 0,
  fontSize: theme.typography.body2.fontSize,
  fontWeight: theme.typography.body2.fontWeight,
  borderRadius: "4px",
  padding: "0px 4px",
  position: "relative",
  left: "-4px",
}));

const ProfileSetting = ({ setting, setSetting }: ProfileSettingProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const nickRef = useRef<HTMLInputElement>(null);

  const user = SessionManager.currentLoginOrNull();
  const [nick, setNick] = useState(user?.user.nickname);
  const [nickLoading, setNickLoading] = useState(false);
  const [profileSettingLoading, setProfileSettingLoading] = useState(false);

  const profileSettingPopup = usePopupState({
    variant: "popover",
    popupId: "profileSetting",
  });

  const profileSettingSummary = useProfileSettingSummary(setting.share_links_in_profile);

  const onClick = () => {
    // Validate input length
    if (nickRef.current && nickRef.current.checkValidity()) {
      setNickLoading(true);
      dispatch(sendUpdateUserSetting({ nick }))
        .then(() => {
          if (user?.user) {
            SessionManager.updateUserIfExist({
              ...user?.user,
              nickname: nick ?? "",
            });
          }
        })
        .finally(() => {
          setNickLoading(false);
        });
    } else {
      // Input is invalid, show validation errors
      nickRef.current?.reportValidity();
    }
  };

  const onProfileSettingChange = (value: ShareLinksInProfileLevel) => {
    setProfileSettingLoading(true);
    dispatch(sendUpdateUserSetting({ share_links_in_profile: value }))
      .then(() => {
        setSetting({
          ...setting,
          share_links_in_profile: value,
        });
        profileSettingPopup.close();
      })
      .finally(() => {
        setProfileSettingLoading(false);
      });
  };

  return (
    <Stack spacing={3}>
      <Grid2
        container
        spacing={isMobile ? 3 : 4}
        direction={isMobile ? "column" : "row-reverse"}
        sx={{ width: "100%" }}
      >
        <Grid2 spacing={3} sx={{ flexGrow: 1, width: "100%" }} size={{ md: 6, xs: 12 }}>
          {user && <AvatarSetting user={user?.user} />}
        </Grid2>
        <Grid2 spacing={3} sx={{ flexGrow: 1, width: "100%" }} size={{ md: 6, xs: 12 }}>
          <Stack spacing={3}>
            <SettingForm title={t("login.email")} noContainer lgWidth={12}>
              <DenseFilledTextField disabled fullWidth value={user?.user.email} />
            </SettingForm>
            <SettingForm title={t("setting.nickname")} noContainer lgWidth={12}>
              <DenseFilledTextField
                fullWidth
                onChange={(e) => setNick(e.target.value)}
                value={nick}
                required
                inputProps={{ maxLength: 255 }}
                helperText={t("setting.nickNameDes")}
                inputRef={nickRef}
              />
              <Collapse in={nick != user?.user.nickname}>
                <LoadingButton variant={"contained"} onClick={onClick} loading={nickLoading} sx={{ mt: 1 }}>
                  <span>{t("fileManager.save")}</span>
                </LoadingButton>
              </Collapse>
            </SettingForm>
            <Grid2 spacing={isMobile ? 3 : 4} container sx={{ width: "100%" }}>
              <SettingForm title={t("setting.uid")} noContainer lgWidth={6}>
                <Typography variant={"body2"} color={"textSecondary"}>
                  {user?.user.id}
                </Typography>
              </SettingForm>
              <SettingForm title={t("setting.regTime")} noContainer lgWidth={6}>
                <Typography variant={"body2"} color={"textSecondary"}>
                  <TimeBadge datetime={user?.user.created_at} variant={"inherit"} />
                </Typography>
              </SettingForm>
            </Grid2>

            <Grid2 spacing={isMobile ? 3 : 4} container sx={{ width: "100%" }}>
              <SettingForm title={t("setting.group")} noContainer lgWidth={6}>
                <Typography variant={"body2"} color={"textSecondary"}>
                  {user?.user.group?.name}
                </Typography>
              </SettingForm>

              <SettingForm title={t("setting.profilePage")} noContainer lgWidth={6}>
                <ProfileDropButton
                  size={"small"}
                  {...bindTrigger(profileSettingPopup)}
                  endIcon={<CaretDown sx={{ fontSize: "12px!important" }} />}
                  disabled={profileSettingLoading}
                >
                  {profileSettingSummary}
                </ProfileDropButton>
                <ProfileSettingPopover
                  currentValue={setting.share_links_in_profile}
                  onValueChange={onProfileSettingChange}
                  {...bindPopover(profileSettingPopup)}
                />
              </SettingForm>
            </Grid2>
          </Stack>
        </Grid2>
      </Grid2>
    </Stack>
  );
};

export default ProfileSetting;
