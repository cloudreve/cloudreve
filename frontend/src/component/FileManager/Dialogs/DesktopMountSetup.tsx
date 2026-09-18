import { DialogContent, Stack, Typography } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { defaultPath } from "../../../hooks/useNavigation.tsx";
import { closeDesktopMountSetupDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { openDesktopCallback } from "../../../redux/thunks/session.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import { PathSelectorForm } from "../../Common/Form/PathSelectorForm.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { useBreadcrumbButtons } from "../TopBar/BreadcrumbButton.tsx";

const DesktopMountSetup = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.desktopMountSetupDialogOpen);
  const callbackState = useAppSelector((state) => state.globalState.desktopMountSetupState);

  const [path, setPath] = useState(defaultPath);

  const uri = useMemo(() => {
    try {
      return new CrUri(path);
    } catch {
      return null;
    }
  }, [path]);

  const pathElements = useMemo(() => uri?.elements() ?? [], [uri]);
  const folderName = pathElements.length > 0 ? pathElements[pathElements.length - 1] : undefined;

  const [_loading, displayName] = useBreadcrumbButtons({
    name: folderName,
    path: path,
    is_root: uri?.is_root(),
  });

  const onClose = useCallback(() => {
    dispatch(closeDesktopMountSetupDialog());
  }, [dispatch]);

  const onAccept = useCallback(() => {
    if (!callbackState || !path || !displayName) {
      return;
    }

    dispatch(openDesktopCallback(callbackState.code, callbackState.state, displayName, path));
  }, [callbackState, path, displayName, dispatch]);

  return (
    <DraggableDialog
      title={t("application:fileManager.desktopMountSetup")}
      showActions
      showCancel
      onAccept={onAccept}
      disabled={!path}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
        disableRestoreFocus: true,
      }}
    >
      <DialogContent>
        <Stack spacing={2} direction={"column"}>
          <Typography variant="body2" color="text.secondary">
            {t("application:fileManager.desktopMountSetupDescription")}
          </Typography>
          <PathSelectorForm
            onChange={setPath}
            path={path}
            variant={"davAccountRoot"}
            label={t("application:setting.rootFolder")}
            allowedFs={[Filesystem.my, Filesystem.share]}
          />
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default DesktopMountSetup;
