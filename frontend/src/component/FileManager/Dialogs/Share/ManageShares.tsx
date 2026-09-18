import {
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
import React, { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getFileInfo, sendDeleteShare } from "../../../../api/api.ts";
import { FileResponse, Share } from "../../../../api/explorer.ts";
import { closeManageShareDialog, setShareLinkDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { confirmOperation } from "../../../../redux/thunks/dialog.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import { NoWrapTableCell, StyledTableContainerPaper } from "../../../Common/StyledComponents.tsx";
import TimeBadge from "../../../Common/TimeBadge.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import MoreVertical from "../../../Icons/MoreVertical.tsx";
import { SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";
import { ShareExpires, ShareStatistics } from "../../TopBar/ShareInfoPopover.tsx";

const ManageShares = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [actionTarget, setActionTarget] = useState<Share | null>(null);
  const [fileExtended, setFileExtended] = useState<FileResponse | undefined>(undefined);
  const [loading, setLoading] = useState(false);

  const open = useAppSelector((state) => state.globalState.manageShareDialogOpen);
  const target = useAppSelector((state) => state.globalState.manageShareDialogFile);

  const onClose = useCallback(() => {
    if (!loading) {
      dispatch(closeManageShareDialog());
    }
  }, [dispatch, loading]);

  useEffect(() => {
    if (target && open) {
      if (target.extended_info) {
        setFileExtended(target);
      } else {
        setFileExtended(undefined);
        dispatch(
          getFileInfo({
            uri: target.path,
            extended: true,
          }),
        ).then((res) => setFileExtended(res));
      }
    }
  }, [target, open]);

  const handleActionClose = () => {
    setAnchorEl(null);
  };

  const handleOpenAction = (event: React.MouseEvent<HTMLElement>, element: Share) => {
    event.stopPropagation();
    setAnchorEl(event.currentTarget);
    setActionTarget(element);
  };

  const openEditDialog = () => {
    dispatch(
      setShareLinkDialog({
        open: true,
        file: target,
        share: actionTarget ?? undefined,
      }),
    );
    setAnchorEl(null);
  };

  const openLink = useCallback((s: Share) => {
    window.open(s.url, "_blank");
  }, []);

  const deleteShare = useCallback(() => {
    if (!target || !actionTarget) {
      return;
    }

    dispatch(confirmOperation(t("fileManager.deleteShareWarning"))).then(() => {
      setLoading(true);
      dispatch(sendDeleteShare(actionTarget.id))
        .then(() => {
          setFileExtended((prev) =>
            prev
              ? {
                  ...prev,
                  extended_info: prev.extended_info
                    ? {
                        ...prev.extended_info,
                        shares: prev.extended_info.shares?.filter((e) => e.id !== actionTarget.id),
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
        <SquareMenuItem dense>
          <ListItemText onClick={openEditDialog}>
            {t(`fileManager.${actionTarget?.expired ? "editAndReactivate" : "edit"}`)}
          </ListItemText>
        </SquareMenuItem>
        <SquareMenuItem dense>
          <ListItemText onClick={deleteShare}>{t(`fileManager.delete`)}</ListItemText>
        </SquareMenuItem>
      </Menu>
      <DraggableDialog
        title={t("application:fileManager.manageShares")}
        loading={loading}
        dialogProps={{
          open: open ?? false,
          onClose: onClose,
          fullWidth: true,
          maxWidth: "md",
        }}
      >
        <DialogContent>
          <AutoHeight>
            <TableContainer component={StyledTableContainerPaper}>
              <Table sx={{ width: "100%" }} size="small">
                <TableHead>
                  <TableRow>
                    <NoWrapTableCell>{t("fileManager.actions")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("fileManager.createdAt")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("fileManager.expires")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("application:share.statistics")}</NoWrapTableCell>
                    <NoWrapTableCell>{t("modals.privateShare")}</NoWrapTableCell>
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
                  {fileExtended?.extended_info?.shares &&
                    fileExtended?.extended_info?.shares.map((e) => (
                      <TableRow
                        sx={{
                          "&:last-child td, &:last-child th": { border: 0 },
                          cursor: "pointer",
                          td: {
                            color: (theme) => (e.expired ? theme.palette.text.disabled : undefined),
                          },
                        }}
                        onClick={() => openLink(e)}
                        hover
                      >
                        <NoWrapTableCell component="th" scope="row">
                          <IconButton disabled={loading} onClick={(event) => handleOpenAction(event, e)} size={"small"}>
                            <MoreVertical fontSize={"small"} />
                          </IconButton>
                        </NoWrapTableCell>
                        <NoWrapTableCell>
                          <TimeBadge variant={"body2"} datetime={e.created_at ?? ""} />
                        </NoWrapTableCell>
                        <TableCell>
                          {e.expired ? (
                            t("application:share.expired")
                          ) : (
                            <>
                              {e.remain_downloads != undefined || e.expires ? (
                                <ShareExpires expires={e.expires} remain_downloads={e.remain_downloads} />
                              ) : (
                                t("application:fileManager.permanentValid")
                              )}
                            </>
                          )}
                        </TableCell>
                        <TableCell>
                          <ShareStatistics shareInfo={e} />
                        </TableCell>
                        <NoWrapTableCell>{t(`fileManager.${e.is_private ? "yes" : "no"}`)}</NoWrapTableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
              {fileExtended && !fileExtended?.extended_info?.shares && (
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
export default ManageShares;
