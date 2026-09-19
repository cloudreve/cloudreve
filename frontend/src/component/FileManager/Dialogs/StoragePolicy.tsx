import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  ListItemText,
  SelectChangeEvent,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getAllowedPolicies, relocateToPolicy, setPreferredPolicy } from "../../../api/api";
import { Metadata, StoragePolicyBrief } from "../../../api/explorer";
import { closeStoragePolicyDialog } from "../../../redux/globalStateSlice";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks";
import { refreshFileList } from "../../../redux/thunks/filemanager";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../ContextMenu/ContextMenu";
import { FileManagerIndex } from "../FileManager";

// StoragePolicyDialog switches the storage policy for either a folder's
// preferred upload policy ("dir") or a file/folder's physical entities
// ("relocate"), choosing from the group's allowed set.
const StoragePolicyDialog = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const open = useAppSelector((state) => state.globalState.storagePolicyDialogOpen);
  const mode = useAppSelector((state) => state.globalState.storagePolicyDialogMode);
  const file = useAppSelector((state) => state.globalState.storagePolicyDialogFile);

  const [policies, setPolicies] = useState<StoragePolicyBrief[]>([]);
  const [value, setValue] = useState<string>("");
  const [loading, setLoading] = useState(false);

  const isDir = mode === "dir";
  const current = file?.metadata?.[Metadata.preferred_policy] ?? "";

  useEffect(() => {
    if (!open) {
      return;
    }
    setValue(isDir ? current : "");
    dispatch(getAllowedPolicies()).then((res) => setPolicies(res ?? []));
  }, [open]);

  const onClose = () => dispatch(closeStoragePolicyDialog());

  const onChange = (e: SelectChangeEvent<unknown>) => {
    setValue(e.target.value as string);
  };

  const onSubmit = () => {
    if (!file) {
      return;
    }
    setLoading(true);
    const uri = file.path;
    const req = isDir
      ? dispatch(setPreferredPolicy({ uri, policy: value })).then(() => {
          enqueueSnackbar(t("fileManager.policySaved"), { variant: "success", action: DefaultCloseAction });
        })
      : dispatch(relocateToPolicy({ uri, policy: value })).then(() => {
          enqueueSnackbar(t("fileManager.relocateSubmitted"), { variant: "success", action: DefaultCloseAction });
        });
    req.then(() => {
      onClose();
      dispatch(refreshFileList(FileManagerIndex.main));
    }).finally(() => setLoading(false));
  };

  return (
    <Dialog open={!!open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{isDir ? t("fileManager.dirPolicyTitle") : t("fileManager.relocateTitle")}</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 2 }}>
          {isDir
            ? t("fileManager.dirPolicyDes", { name: file?.name ?? "" })
            : t("fileManager.relocateDes", { name: file?.name ?? "" })}
        </DialogContentText>
        <FormControl fullWidth>
          <DenseSelect value={value} onChange={onChange} disabled={loading}>
            {isDir && (
              <SquareMenuItem value="">
                <ListItemText primary={t("fileManager.policyInherit")} />
              </SquareMenuItem>
            )}
            {policies.map((p) => (
              <SquareMenuItem key={p.id} value={p.id}>
                <ListItemText
                  primary={p.name}
                  secondary={p.is_default ? t("fileManager.policyGroupDefault") : undefined}
                />
              </SquareMenuItem>
            ))}
          </DenseSelect>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button
          variant="contained"
          onClick={onSubmit}
          disabled={loading || (!isDir && !value) || (isDir && value === current)}
        >
          {isDir ? t("common:save") : t("fileManager.relocateStart")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default StoragePolicyDialog;
