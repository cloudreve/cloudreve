import { Alert, AlertTitle, Checkbox, DialogContent, Stack, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendImport } from "../../../api/api";
import { User } from "../../../api/user";
import { useAppDispatch } from "../../../redux/hooks";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";
import { DenseFilledTextField, SmallFormControlLabel } from "../../Common/StyledComponents";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import SettingForm from "../../Pages/Setting/SettingForm";
import SinglePolicySelectionInput from "../Common/SinglePolicySelectionInput";
import { NoMarginHelperText } from "../Settings/Settings";
import UserSearchInput from "./UserSearchInput";
// TODO: Add API call for creating import task

export interface ImportFileDialogProps {
  open: boolean;
  onClose: () => void;
}

interface ImportTaskForm {
  storagePolicyId: number | undefined;
  externalPath: string;
  recursive: boolean;
  targetUser: User | null;
  targetPath: string;
  extractMediaMeta: boolean | undefined;
}

const defaultForm: ImportTaskForm = {
  storagePolicyId: undefined,
  externalPath: "",
  recursive: true,
  targetUser: null,
  targetPath: "/",
  extractMediaMeta: false,
};

export const ImportFileDialog = ({ open, onClose }: ImportFileDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(false);
  const [formState, setFormState] = useState<ImportTaskForm>({ ...defaultForm });
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (open) {
      setFormState({ ...defaultForm });
    }
  }, [open]);

  const handleSubmit = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }

    if (!formState.targetUser) {
      enqueueSnackbar(t("file.pleaseSelectUser"), { variant: "error", action: DefaultCloseAction });
      return;
    }

    setLoading(true);
    dispatch(
      sendImport({
        src: formState.externalPath,
        dst: formState.targetPath,
        extract_media_meta: false,
        user_id: formState.targetUser.id,
        recursive: formState.recursive,
        policy_id: formState.storagePolicyId ?? 0,
      }),
    )
      .then(() => {
        enqueueSnackbar(t("file.importTaskCreated"), { variant: "success", action: DefaultCloseAction });
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const handlePolicyChange = (value: number) => {
    setFormState({ ...formState, storagePolicyId: value });
  };

  const handleUserSelected = (user: User) => {
    setFormState({ ...formState, targetUser: user });
  };

  return (
    <DraggableDialog
      onAccept={handleSubmit}
      loading={loading}
      title={t("file.importExternalFolder")}
      showActions
      showCancel
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <form
          ref={formRef}
          onSubmit={(e) => {
            e.preventDefault();
            handleSubmit();
          }}
        >
          <Stack spacing={2}>
            <Alert severity="info" sx={{ fontSize: (theme) => theme.typography.body2.fontSize }}>
              {t("file.importExternalFolderDes")}
            </Alert>
            <Alert severity="warning" sx={{ fontSize: (theme) => theme.typography.body2.fontSize }}>
              <AlertTitle>{t("file.importWarning")}</AlertTitle>
              <ul style={{ paddingInlineStart: "20px" }}>
                {t("file.importWarnings", { returnObjects: true }).map((warning, index) => (
                  <li key={index}>{warning.toString()}</li>
                ))}
              </ul>
            </Alert>
            <SettingForm lgWidth={12} title={t("file.storagePolicy")}>
              <SinglePolicySelectionInput value={formState.storagePolicyId} onChange={handlePolicyChange} />
              <NoMarginHelperText>{t("file.storagePolicyDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm lgWidth={12} title={t("file.srcFolderPath")}>
              <DenseFilledTextField
                fullWidth
                required
                placeholder={"/path/to/folder"}
                value={formState.externalPath}
                onChange={(e) => setFormState({ ...formState, externalPath: e.target.value })}
              />
              <NoMarginHelperText>{t("file.selectSrcDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm lgWidth={12} title={t("file.targetUser")}>
              <UserSearchInput onUserSelected={handleUserSelected} label={t("file.searchUser")} required />
              <NoMarginHelperText>{t("file.targetUserDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm lgWidth={12} title={t("file.dstFolderPath")}>
              <DenseFilledTextField
                fullWidth
                required
                value={formState.targetPath}
                onChange={(e) => setFormState({ ...formState, targetPath: e.target.value })}
                placeholder={"/"}
              />
              <NoMarginHelperText>{t("file.dstFolderPathDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm lgWidth={12}>
              <SmallFormControlLabel
                control={
                  <Checkbox
                    size={"small"}
                    checked={formState.recursive}
                    onChange={(e) => setFormState({ ...formState, recursive: e.target.checked })}
                  />
                }
                labelPlacement="end"
                label={<Typography variant="body2">{t("file.recursivelyImport")}</Typography>}
              />
              <NoMarginHelperText>{t("file.recursivelyImportDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm lgWidth={12}>
              <SmallFormControlLabel
                control={
                  <Checkbox
                    size={"small"}
                    checked={formState.extractMediaMeta}
                    onChange={(e) => setFormState({ ...formState, extractMediaMeta: e.target.checked })}
                  />
                }
                labelPlacement="end"
                label={<Typography variant="body2">{t("file.extractMediaMeta")}</Typography>}
              />
              <NoMarginHelperText>{t("file.extractMediaMetaDes")}</NoMarginHelperText>
            </SettingForm>
          </Stack>
        </form>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ImportFileDialog;
