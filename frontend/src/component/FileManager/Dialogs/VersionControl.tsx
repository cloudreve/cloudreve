import {
  Alert,
  Box,
  DialogContent,
  IconButton,
  ListItemText,
  Menu,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { confirmOperation } from "../../../redux/thunks/dialog.ts";
import { downloadSingleFile } from "../../../redux/thunks/download.ts";
import { setFileVersion } from "../../../redux/thunks/file.ts";
import { openViewers } from "../../../redux/thunks/viewer.ts";
import { sizeToString } from "../../../util";
import AutoHeight from "../../Common/AutoHeight.tsx";
import { closeVersionControlDialog } from "../../../redux/globalStateSlice.ts";
import { Entity, EntityType, FileResponse } from "../../../api/explorer.ts";
import { deleteVersion, getFileInfo } from "../../../api/api.ts";
import { NoWrapTableCell, StyledTableContainerPaper } from "../../Common/StyledComponents.tsx";
import TimeBadge from "../../Common/TimeBadge.tsx";
import { AnonymousUser } from "../../Common/User/UserAvatar.tsx";
import UserBadge from "../../Common/User/UserBadge.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import MoreVertical from "../../Icons/MoreVertical.tsx";
import { SquareMenuItem } from "../ContextMenu/ContextMenu.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { EncryptionStatusText } from "../Sidebar/BasicInfo.tsx";

const VersionControl = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [actionTarget, setActionTarget] = useState<Entity | null>(null);
  const [fileExtended, setFileExtended] = useState<FileResponse | undefined>(undefined);
  const [loading, setLoading] = useState(false);

  const open = useAppSelector((state) => state.globalState.versionControlDialogOpen);
  const target = useAppSelector((state) => state.globalState.versionControlDialogFile);
  const highlight = useAppSelector((state) => state.globalState.versionControlHighlight);
  const showEncryptionStatus = useAppSelector((state) => state.siteConfig?.explorer?.config?.show_encryption_status);

  const onClose = useCallback(() => {
    if (!loading) {
      dispatch(closeVersionControlDialog());
    }
  }, [dispatch, loading]);

  useEffect(() => {
    if (target && open) {
      setFileExtended(undefined);
      dispatch(
        getFileInfo({
          uri: target.path,
          extended: true,
        }),
      ).then((res) => setFileExtended(res));
    }
  }, [target, open]);

  const versionEntities = useMemo(() => {
    return fileExtended?.extended_info?.entities?.filter((e) => e.type == EntityType.version);
  }, [fileExtended?.extended_info?.entities]);

  const hilightButNotFound = useMemo(() => {
    return highlight && fileExtended?.extended_info && !versionEntities?.some((e) => e.id == highlight);
  }, [highlight, fileExtended?.extended_info?.entities]);

  const handleActionClose = () => {
    setAnchorEl(null);
  };

  const handleOpenAction = (event: React.MouseEvent<HTMLElement>, element: Entity) => {
    setAnchorEl(event.currentTarget);
    setActionTarget(element);
  };

  const downloadEntity = useCallback(() => {
    if (!target || !actionTarget) {
      return;
    }
    dispatch(downloadSingleFile(target, actionTarget.id));
    setAnchorEl(null);
  }, [target, actionTarget, dispatch]);

  const openEntity = useCallback(() => {
    if (!target || !actionTarget) {
      return;
    }
    dispatch(openViewers(FileManagerIndex.main, target, actionTarget.size, actionTarget.id));
    setAnchorEl(null);
  }, [target, actionTarget, dispatch]);

  const setAsCurrent = useCallback(() => {
    if (!target || !actionTarget) {
      return;
    }

    setLoading(true);
    dispatch(setFileVersion(FileManagerIndex.main, target, actionTarget.id))
      .then(() => {
        setFileExtended((prev) =>
          prev
            ? {
                ...prev,
                primary_entity: actionTarget.id,
              }
            : undefined,
        );
      })
      .finally(() => {
        setLoading(false);
      });

    setAnchorEl(null);
  }, [target, actionTarget, setLoading, dispatch]);

  const deleteTargetVersion = useCallback(() => {
    if (!target || !actionTarget) {
      return;
    }

    dispatch(confirmOperation(t("fileManager.deleteVersionWarning"))).then(() => {
      setLoading(true);
      dispatch(
        deleteVersion({
          uri: target.path,
          version: actionTarget.id,
        }),
      )
        .then(() => {
          setFileExtended((prev) =>
            prev
              ? {
                  ...prev,
                  extended_info: prev.extended_info
                    ? {
                        ...prev.extended_info,
                        entities: prev.extended_info.entities?.filter((e) => e.id !== actionTarget.id),
                      }
                    : undefined,
                }
              : undefined,
          );
        })
        .finally(() => {
          setLoading(false);
        });
    });

    setAnchorEl(null);
  }, [t, target, actionTarget, setLoading, dispatch]);

  return (
    <>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleActionClose}
        slotProps={{
          paper: {
            sx: {
              minWidth: 150,
            },
          },
        }}
      >
        <SquareMenuItem onClick={openEntity} dense>
          <ListItemText>{t("application:fileManager.open")}</ListItemText>
        </SquareMenuItem>
        <SquareMenuItem onClick={downloadEntity} dense>
          <ListItemText>{t("application:fileManager.download")}</ListItemText>
        </SquareMenuItem>
        {target?.owned && actionTarget?.id !== fileExtended?.primary_entity && (
          <SquareMenuItem onClick={setAsCurrent} dense>
            <ListItemText>{t("application:fileManager.setAsCurrent")}</ListItemText>
          </SquareMenuItem>
        )}
        {target?.owned && actionTarget?.id !== fileExtended?.primary_entity && (
          <SquareMenuItem onClick={deleteTargetVersion} dense>
            <ListItemText>{t("application:fileManager.delete")}</ListItemText>
          </SquareMenuItem>
        )}
      </Menu>
      <DraggableDialog
        title={t("application:fileManager.manageVersions")}
        loading={loading}
        dialogProps={{
          open: open ?? false,
          onClose: onClose,
          fullWidth: true,
          maxWidth: "sm",
        }}
      >
        <DialogContent>
          <AutoHeight>
            {hilightButNotFound && (
              <Alert severity="warning" sx={{ mb: 2 }}>
                {t("application:fileManager.versionNotFound")}
              </Alert>
            )}
            <TableContainer component={StyledTableContainerPaper}>
              <Table sx={{ width: "100%" }} size="small">
                <TableHead>
                  <TableRow>
                    <NoWrapTableCell>{t("fileManager.actions")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("fileManager.createdAt")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("fileManager.size")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("fileManager.createdBy")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("application:fileManager.storagePolicy")}</NoWrapTableCell>
                    {showEncryptionStatus && (
                      <NoWrapTableCell>{t("application:fileManager.encryption")}</NoWrapTableCell>
                    )}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {!fileExtended && (
                    <TableRow
                      hover
                      sx={{
                        "&:last-child td, &:last-child th": { border: 0 },
                      }}
                    >
                      <NoWrapTableCell component="th" scope="row">
                        <Skeleton variant={"text"} width={100} />
                      </NoWrapTableCell>
                      <NoWrapTableCell>
                        <Skeleton variant={"text"} width={30} />
                      </NoWrapTableCell>
                    </TableRow>
                  )}
                  {versionEntities &&
                    versionEntities.map((e) => (
                      <TableRow
                        selected={e.id === fileExtended?.primary_entity}
                        sx={{
                          boxShadow: (theme) =>
                            highlight == e.id ? `inset 0 0 0 2px ${theme.palette.primary.light}` : "none",
                          "&:last-child td, &:last-child th": { border: 0 },
                        }}
                        hover
                      >
                        <NoWrapTableCell component="th" scope="row">
                          <IconButton disabled={loading} onClick={(event) => handleOpenAction(event, e)} size={"small"}>
                            <MoreVertical fontSize={"small"} />
                          </IconButton>
                        </NoWrapTableCell>
                        <NoWrapTableCell>
                          <TimeBadge variant={"body2"} datetime={e.created_at} />
                        </NoWrapTableCell>
                        <NoWrapTableCell>{sizeToString(e.size)}</NoWrapTableCell>
                        <TableCell>
                          <UserBadge
                            sx={{ width: 20, height: 20 }}
                            textProps={{
                              variant: "body2",
                            }}
                            user={e.created_by ?? AnonymousUser}
                          />
                        </TableCell>
                        <NoWrapTableCell>{e.storage_policy?.name}</NoWrapTableCell>
                        {showEncryptionStatus && (
                          <NoWrapTableCell>
                            <EncryptionStatusText
                              flexWrap={false}
                              status={{
                                status: e.encrypted_with ? "full" : "none",
                                cipher: e.encrypted_with ? [e.encrypted_with] : [],
                              }}
                            />
                          </NoWrapTableCell>
                        )}
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
              {!versionEntities && fileExtended && (
                <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
                  <Typography variant={"caption"} color={"text.secondary"}>
                    {t("application:setting.listEmpty")}
                  </Typography>
                </Box>
              )}
            </TableContainer>
          </AutoHeight>
        </DialogContent>
      </DraggableDialog>
    </>
  );
};
export default VersionControl;
