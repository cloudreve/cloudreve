import { Box, Grid2 as Grid, Link, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { Entity } from "../../../../api/dashboard";
import { EntityType } from "../../../../api/explorer";
import { useAppDispatch } from "../../../../redux/hooks";
import { sizeToString } from "../../../../util";
import { NoWrapTypography } from "../../../Common/StyledComponents";
import UserAvatar from "../../../Common/User/UserAvatar";
import { EncryptionStatusText } from "../../../FileManager/Sidebar/BasicInfo";
import { EntityTypeText } from "../../../FileManager/Sidebar/Data";
import SettingForm from "../../../Pages/Setting/SettingForm";
import UserDialog from "../../User/UserDialog/UserDialog";
import EntityFileList from "./EntityFileList";
const EntityForm = ({ values }: { values: Entity }) => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const { t } = useTranslation("dashboard");
  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number>(0);

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    setUserDialogOpen(true);
    setUserDialogID(values?.edges?.user?.id ?? 0);
  };

  return (
    <>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <Box>
        <Grid container spacing={isMobile ? 2 : 3} alignItems={"stretch"}>
          <SettingForm title={t("file.id")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values.id}
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.size")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {sizeToString(values.size ?? 0)}
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.blobType")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {t(EntityTypeText[values.type ?? EntityType.version])}
            </Typography>
          </SettingForm>
          <SettingForm title={t("entity.refenenceCount")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values.reference_count ?? 0}
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
          <SettingForm title={t("entity.uploadSessionID")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"} sx={{ wordBreak: "break-all" }}>
              {values.upload_session_id ?? "-"}
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.source")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"} sx={{ wordBreak: "break-all" }}>
              {values.source ?? "-"}
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.storagePolicy")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              <Link
                component={RouterLink}
                underline="hover"
                to={`/admin/policy/${values.edges?.storage_policy?.id}`}
                target="_blank"
              >
                {values.edges?.storage_policy?.name}
              </Link>
            </Typography>
          </SettingForm>
          <SettingForm title={t("application:fileManager.encryption")} noContainer lgWidth={4}>
            <EncryptionStatusText
              status={{
                status: values.props?.encrypt_metadata?.algorithm ? "full" : "none",
                cipher: values.props?.encrypt_metadata?.algorithm ? [values.props?.encrypt_metadata?.algorithm] : [],
              }}
            />
          </SettingForm>
          <SettingForm title={t("entity.referredFiles")} noContainer lgWidth={12}>
            <EntityFileList files={values.edges?.file ?? []} userHashIDMap={values.user_hash_id_map ?? {}} />
          </SettingForm>
        </Grid>
      </Box>
    </>
  );
};

export default EntityForm;
