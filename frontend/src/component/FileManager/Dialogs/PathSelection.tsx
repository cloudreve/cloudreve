import { DialogContent, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { FileResponse, FileType } from "../../../api/explorer.ts";
import { closePathSelectionDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { pathSelectionDialogPromisePool } from "../../../redux/thunks/dialog.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import FileBadge from "../FileBadge.tsx";
import FolderPicker, { useFolderSelector } from "../FolderPicker.tsx";

export const PathSelectionVariantOptions = {
  copy: "copy",
  move: "move",
  shortcut: "shortcut",
};

interface SelectedFolderIndicatorProps {
  selectedFile?: FileResponse;
  selectedPath?: string;
  variant: PathSelectionVariant;
}

interface PathSelectionVariant {
  indicator: string;
  title: string;
  disableSharedWithMe?: boolean;
  disableTrash?: boolean;
}

export const PathSelectionVariants: Record<string, PathSelectionVariant> = {
  copy: {
    indicator: "fileManager.copyToDst",
    title: "application:fileManager.copyTo",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  move: {
    indicator: "fileManager.moveToDst",
    title: "application:fileManager.moveTo",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  shortcut: {
    indicator: "application:modals.createShortcutTo",
    title: "application:modals.createShortcut",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  saveAs: {
    indicator: "application:modals.saveToTitleDescription",
    title: "application:modals.saveAs",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  saveTo: {
    indicator: "application:modals.saveToTitleDescription",
    title: "application:modals.saveToTitle",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  extractTo: {
    indicator: "application:modals.decompressToDst",
    title: "application:modals.decompressTo",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  downloadTo: {
    indicator: "application:modals.downloadToDst",
    title: "application:modals.downloadTo",
    disableSharedWithMe: true,
    disableTrash: true,
  },
  searchIn: {
    indicator: "application:navbar.searchInBase",
    title: "application:navbar.searchBase",
  },
  davAccountRoot: {
    indicator: "application:setting.rootFolderIn",
    title: "application:setting.rootFolder",
    disableSharedWithMe: true,
    disableTrash: true,
  },
};

export const SelectedFolderIndicator = ({ selectedFile, selectedPath, variant }: SelectedFolderIndicatorProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  if (!selectedFile && !selectedPath) {
    return null;
  }

  const badge = (
    <FileBadge
      file={selectedFile}
      variant={"outlined"}
      sx={{ mx: 1, maxWidth: isMobile ? "150px" : "initial" }}
      simplifiedFile={
        selectedPath
          ? {
              path: selectedPath,
              type: FileType.folder,
            }
          : undefined
      }
    />
  );
  return (
    <Typography variant={"body2"} color={"text.secondary"}>
      {isMobile ? badge : <Trans i18nKey={variant.indicator} ns={"application"} components={[badge]} />}
    </Typography>
  );
};

const PathSelection = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [loading, setLoading] = useState(false);

  const open = useAppSelector((state) => state.globalState.pathSelectDialogOpen);
  const variant = useAppSelector((state) => state.globalState.pathSelectDialogVariant);
  const promiseId = useAppSelector((state) => state.globalState.pathSelectPromiseId);
  const initialPath = useAppSelector((state) => state.globalState.pathSelectInitialPath);

  const variantProps = useMemo(
    () => (variant ? PathSelectionVariants[variant] : PathSelectionVariants["copy"]),
    [variant],
  );

  const [selectedFile, selectedPath] = useFolderSelector();

  const onClose = useCallback(() => {
    dispatch(closePathSelectionDialog());
    if (promiseId) {
      pathSelectionDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onAccept = useCallback(async () => {
    const dst = selectedPath;

    dispatch(closePathSelectionDialog());
    if (promiseId && dst) {
      pathSelectionDialogPromisePool[promiseId]?.resolve(dst);
    }
  }, [dispatch, selectedPath, promiseId]);

  const disabled = useMemo(() => {
    const dst = selectedPath;
    if (dst) {
      const crUri = new CrUri(dst);
      if (variantProps.disableSharedWithMe && crUri.fs() == Filesystem.shared_with_me) {
        return true;
      }
      if (variantProps.disableTrash && crUri.fs() == Filesystem.trash) {
        return true;
      }
    }

    return !selectedPath;
  }, [selectedPath, variantProps]);

  return (
    <DraggableDialog
      title={t(variantProps.title)}
      showActions
      loading={loading}
      disabled={disabled}
      secondaryAction={
        <SelectedFolderIndicator variant={variantProps} selectedFile={selectedFile} selectedPath={selectedPath} />
      }
      onAccept={onAccept}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "lg",
        disableRestoreFocus: true,
        PaperProps: {
          sx: {
            height: "100%",
          },
        },
      }}
    >
      <DialogContent sx={{ display: "flex", pb: 0 }}>
        <FolderPicker
          disableSharedWithMe={variantProps.disableSharedWithMe}
          disableTrash={variantProps.disableTrash}
          initialPath={initialPath}
        />
      </DialogContent>
    </DraggableDialog>
  );
};
export default PathSelection;
