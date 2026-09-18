import { Box, Grid2 as Grid, Link, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Share } from "../../../../api/dashboard";
import { NoWrapTypography } from "../../../Common/StyledComponents";
import TimeBadge from "../../../Common/TimeBadge";
import UserAvatar from "../../../Common/User/UserAvatar";
import FileTypeIcon from "../../../FileManager/Explorer/FileTypeIcon";
import SettingForm from "../../../Pages/Setting/SettingForm";
import FileDialog from "../../File/FileDialog/FileDialog";
import UserDialog from "../../User/UserDialog/UserDialog";

const ShareForm = ({ values }: { values: Share }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const { t } = useTranslation("dashboard");
  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number>(0);
  const [fileDialogOpen, setFileDialogOpen] = useState(false);
  const [fileDialogID, setFileDialogID] = useState<number>(0);

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    setUserDialogOpen(true);
    setUserDialogID(values?.edges?.user?.id ?? 0);
  };

  const fileClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    setFileDialogOpen(true);
    setFileDialogID(values?.edges?.file?.id ?? 0);
  };

  return (
    <>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <FileDialog open={fileDialogOpen} onClose={() => setFileDialogOpen(false)} fileID={fileDialogID} />
      <Box>
        <Grid container spacing={isMobile ? 2 : 3} alignItems={"stretch"}>
          <SettingForm title={t("file.id")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values.id}
            </Typography>
          </SettingForm>

          <SettingForm title={t("share.shareLink")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              <Link href={values.share_link} target="_blank" underline="hover" sx={{ wordBreak: "break-all" }}>
                {values.share_link}
              </Link>
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.creator")} noContainer lgWidth={4}>
            <NoWrapTypography variant={"body2"} color={"textSecondary"}>
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <UserAvatar
                  sx={{ width: 24, height: 24 }}
                  overwriteTextSize
                  user={{
                    id: values?.user_hash_id ?? "",
                    nickname: values?.edges?.user?.nick ?? "",
                    created_at: values?.edges?.user?.created_at ?? "",
                  }}
                />
                <NoWrapTypography variant="inherit">
                  <Link
                    sx={{
                      whiteSpace: "nowrap",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    }}
                    onClick={userClicked}
                    underline="hover"
                    href="#/"
                  >
                    {values?.edges?.user?.nick}
                  </Link>
                </NoWrapTypography>
              </Box>
            </NoWrapTypography>
          </SettingForm>

          <SettingForm title={t("share.views")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values.views ?? 0}
            </Typography>
          </SettingForm>

          <SettingForm title={t("share.downloads")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values.downloads ?? 0}
            </Typography>
          </SettingForm>

          <SettingForm title={t("share.srcFileName")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values?.edges?.file ? (
                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                  <FileTypeIcon name={values?.edges?.file?.name ?? ""} fileType={values?.edges?.file?.type ?? 0} />
                  <NoWrapTypography variant="inherit">
                    <Link href={`#/`} onClick={fileClicked} underline="hover">
                      {values?.edges?.file?.name}
                    </Link>
                  </NoWrapTypography>
                </Box>
              ) : (
                <em>{t("share.deleted")}</em>
              )}
            </Typography>
          </SettingForm>
          <SettingForm title={t("share.createdAt")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              <TimeBadge datetime={values.created_at ?? ""} timeAgoThreshold={0} variant="inherit" />
            </Typography>
          </SettingForm>
        </Grid>
      </Box>
    </>
  );
};

export default ShareForm;
