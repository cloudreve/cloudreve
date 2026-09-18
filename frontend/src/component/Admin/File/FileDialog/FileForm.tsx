import { Box, Grid2 as Grid, Link, Skeleton, styled, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { getFileInfo } from "../../../../api/api";
import { FileType } from "../../../../api/explorer";
import { useAppDispatch } from "../../../../redux/hooks";
import { sizeToString } from "../../../../util";
import CrUri from "../../../../util/uri";
import { DenseFilledTextField, NoWrapTypography } from "../../../Common/StyledComponents";
import UserAvatar from "../../../Common/User/UserAvatar";
import FileBadge from "../../../FileManager/FileBadge";
import SettingForm from "../../../Pages/Setting/SettingForm";
import SinglePolicySelectionInput from "../../Common/SinglePolicySelectionInput";
import UserDialog from "../../User/UserDialog/UserDialog";
import { FileDialogContext } from "./FileDialog";
import FileDirectLinks from "./FileDirectLinks";
import FileEntity from "./FileEntity";
import FileMetadata from "./FileMetadata";

const StyledFileBadge = styled(FileBadge)(({ theme }) => ({
  minHeight: "40px",
  border: `1px solid ${theme.palette.mode === "light" ? "rgba(0, 0, 0, 0.23)" : "rgba(255, 255, 255, 0.23)"}`,
}));

const FileForm = () => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const { t } = useTranslation("dashboard");
  const { formRef, values, setFile } = useContext(FileDialogContext);
  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number>(0);
  const [fileParentLoading, setFileParentLoading] = useState(true);
  const [fileParent, setFileParent] = useState<string | null>(null);

  useEffect(() => {
    setFileParentLoading(true);
    dispatch(getFileInfo({ id: values.file_hash_id }, true))
      .then((res) => {
        const crUri = new CrUri(res.path);
        setFileParent(crUri.parent().toString());
      })
      .catch(() => {
        setFileParent(null);
      })
      .finally(() => {
        setFileParentLoading(false);
      });
  }, [values.id]);

  const onNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFile((prev) => ({ ...prev, name: e.target.value }));
    },
    [setFile],
  );

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    setUserDialogOpen(true);
    setUserDialogID(values?.edges?.owner?.id ?? 0);
  };

  const sizeUsed = useMemo(() => {
    return sizeToString(values?.edges?.entities?.reduce((acc, entity) => acc + (entity.size ?? 0), 0) ?? 0);
  }, [values?.edges?.entities]);

  return (
    <>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
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
          <SettingForm title={t("file.sizeUsed")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {sizeUsed}
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.shareLink")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              <Trans
                i18nKey="file.shareLinkNum"
                components={[
                  <Link component={RouterLink} to={`/admin/share?file=${values?.id}`} underline={"hover"} key={0} />,
                ]}
                ns="dashboard"
                values={{ num: values.edges?.shares?.length ?? 0 }}
              />
            </Typography>
          </SettingForm>
          <SettingForm title={t("file.uploader")} noContainer lgWidth={4}>
            <NoWrapTypography variant={"body2"} color={"textSecondary"}>
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <UserAvatar
                  sx={{ width: 24, height: 24 }}
                  overwriteTextSize
                  user={{
                    id: values?.user_hash_id ?? "",
                    nickname: values?.edges?.owner?.nick ?? "",
                    created_at: values?.edges?.owner?.created_at ?? "",
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
                    {values?.edges?.owner?.nick}
                  </Link>
                </NoWrapTypography>
              </Box>
            </NoWrapTypography>
          </SettingForm>
          <SettingForm title={t("file.name")} noContainer lgWidth={6}>
            <DenseFilledTextField fullWidth value={values.name} required onChange={onNameChange} />
          </SettingForm>
          <SettingForm title={t("application:fileManager.parentFolder")} noContainer lgWidth={3}>
            {!fileParentLoading && (
              <>
                {fileParent && (
                  <StyledFileBadge variant={"outlined"} simplifiedFile={{ path: fileParent, type: FileType.folder }} />
                )}
                {!fileParent && "-"}
              </>
            )}
            {fileParentLoading && <Skeleton variant="text" width={100} height={40} />}
          </SettingForm>
          <SettingForm title={t("file.primaryStoragePolicy")} noContainer lgWidth={3} pro>
            <SinglePolicySelectionInput simplified value={values.storage_policy_files ?? 0} onChange={() => {}} />
          </SettingForm>
          <SettingForm title={t("file.metadata")} noContainer lgWidth={6}>
            <FileMetadata />
          </SettingForm>
          <SettingForm title={t("file.blobs")} noContainer lgWidth={6}>
            <FileEntity />
          </SettingForm>
          <SettingForm title={t("file.directLinks")} noContainer lgWidth={12}>
            <FileDirectLinks />
          </SettingForm>
        </Grid>
      </Box>
    </>
  );
};

export default FileForm;
