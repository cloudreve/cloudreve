import {
  Box,
  Button,
  DialogContent,
  Stack,
  styled,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ConflictDetail, FileResponse, LockApplication } from "../../../api/explorer.ts";
import { closeLockConflictDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { ViewersByID } from "../../../redux/siteConfigSlice.ts";
import { generalDialogPromisePool } from "../../../redux/thunks/dialog.ts";
import { forceUnlockFiles } from "../../../redux/thunks/file.ts";
import { NoWrapTableCell, StyledTableContainerPaper } from "../../Common/StyledComponents.tsx";
import DraggableDialog, { StyledDialogActions, StyledDialogContentText } from "../../Dialogs/DraggableDialog.tsx";
import FileBadge from "../FileBadge.tsx";
import { ViewerIcon } from "./OpenWith.tsx";

interface ErrorTableProps {
  data: ConflictDetail[];
  loading?: boolean;
  files: {
    [key: string]: FileResponse;
  };
  unlock: (tokens: string[]) => Promise<void>;
}

export const CellHeaderWithPadding = styled(Box)({
  paddingLeft: "8px",
});

const ErrorTable = (props: ErrorTableProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  return (
    <TableContainer component={StyledTableContainerPaper}>
      <Table sx={{ width: "100%" }} size="small">
        <TableHead>
          <TableRow>
            <TableCell>
              <CellHeaderWithPadding>{t("common:object")}</CellHeaderWithPadding>
            </TableCell>
            <TableCell>{t("application:modals.application")}</TableCell>
            <TableCell>
              <CellHeaderWithPadding>{t("application:setting.action")}</CellHeaderWithPadding>
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {props.data.map((conflict, i) => (
            <TableRow hover key={i}>
              <TableCell component="th" scope="row">
                {conflict.path && (
                  <FileBadge
                    sx={{ maxWidth: "250px" }}
                    simplifiedFile={{
                      path: conflict.path ?? "",
                      type: conflict.type,
                    }}
                    file={props.files[conflict.path ?? ""]}
                  />
                )}
                {!conflict.path && <FileBadge sx={{ maxWidth: "250px" }} unknown />}
              </TableCell>
              <NoWrapTableCell>
                {conflict.owner?.application && <Application app={conflict.owner?.application} />}
              </NoWrapTableCell>
              <NoWrapTableCell>
                <Tooltip title={!conflict.token ? t("application:modals.onlyOwner") : ""}>
                  <span>
                    <Button
                      disabled={!conflict.token || props.loading}
                      onClick={() => props.unlock([conflict.token ?? ""])}
                    >
                      <Typography variant={"body2"}>{t("application:modals.forceUnlock")}</Typography>
                    </Button>
                  </span>
                </Tooltip>
              </NoWrapTableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {(!props.data || props.data.length === 0) && (
        <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
          <Typography variant={"caption"} color={"text.secondary"}>
            {t("application:setting.listEmpty")}
          </Typography>
        </Box>
      )}
    </TableContainer>
  );
};

interface ApplicationProps {
  app: LockApplication;
}

const ApplicationNameMap: {
  [key: string]: string;
} = {
  rename: "application:fileManager.rename",
  moveCopy: "application:modals.moveCopy",
  upload: "application:modals.upload",
  updateMetadata: "application:modals.updateMetadata",
  delete: "application:fileManager.delete",
  softDelete: "application:fileManager.delete",
  dav: "application:modals.webdav",
  versionControl: "fileManager.manageVersions",
};

const viewerType = "viewer";

const Application = ({ app }: ApplicationProps) => {
  const { t } = useTranslation();
  const title = ApplicationNameMap[app.type] ?? app.type;
  if (app.type == "viewer" && ViewersByID[app.viewer_id ?? ""]) {
    const viewer = ViewersByID[app.viewer_id ?? ""];
    if (viewer) {
      return (
        <Box sx={{ display: "flex", alignItems: "center" }}>
          <Box sx={{ mr: 1 }}>
            <ViewerIcon size={20} viewer={viewer} />
          </Box>
          {viewer?.display_name}
        </Box>
      );
    }
  }
  return <Box sx={{ display: "flex", alignItems: "center" }}>{t(title)}</Box>;
};

const LockConflictDetails = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.lockConflictDialogOpen);
  const files = useAppSelector((state) => state.globalState.lockConflictFile);
  const error = useAppSelector((state) => state.globalState.lockConflictError);
  const promiseId = useAppSelector((state) => state.globalState.lockConflictPromiseId);

  const [loading, setLoading] = useState(false);

  const onClose = useCallback(() => {
    dispatch(closeLockConflictDialog());
    if (promiseId) {
      generalDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onRetry = useCallback(() => {
    if (promiseId) {
      dispatch(closeLockConflictDialog());
      generalDialogPromisePool[promiseId]?.resolve();
    }
  }, [promiseId]);

  const showUnlockAll = useMemo(() => {
    if (error && error.data) {
      for (const conflict of error.data) {
        if (conflict.token) {
          return true;
        }
      }
    }
    return false;
  }, [error]);

  const forceUnlockByToken = useCallback(
    async (tokens: string[]) => {
      setLoading(true);
      try {
        await dispatch(forceUnlockFiles(tokens));
      } finally {
        setLoading(false);
      }
    },
    [dispatch, setLoading],
  );

  const unlockAll = useCallback(async () => {
    const tokens = error?.data?.filter((c) => c.token).map((c) => c.token ?? "");
    if (tokens) {
      await forceUnlockByToken(tokens);
    }
  }, [forceUnlockByToken, error]);

  return (
    <DraggableDialog
      title={t("application:modals.lockConflictTitle")}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <Stack spacing={2}>
          <StyledDialogContentText>{t("application:modals.lockConflictDescription")}</StyledDialogContentText>
          {files && error && error.data && (
            <ErrorTable unlock={forceUnlockByToken} loading={loading} data={error.data} files={files} />
          )}
          {showUnlockAll && (
            <Box>
              <Button onClick={unlockAll} disabled={loading} variant={"contained"}>
                {t("application:modals.forceUnlockAll")}
              </Button>
            </Box>
          )}
        </Stack>
      </DialogContent>
      <StyledDialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button variant={"contained"} disabled={loading} onClick={onRetry}>
          {t("application:uploader.retry")}
        </Button>
      </StyledDialogActions>
    </DraggableDialog>
  );
};

export default LockConflictDetails;
