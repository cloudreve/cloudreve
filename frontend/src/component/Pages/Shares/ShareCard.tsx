import {
  Box,
  Chip,
  Grid,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuProps,
  Skeleton,
  Tooltip,
  Typography,
  useTheme,
} from "@mui/material";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useSnackbar } from "notistack";
import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useInView } from "react-intersection-observer";
import { useNavigate } from "react-router-dom";
import { sendDeleteShare } from "../../../api/api.ts";
import { FileType, Share } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { confirmOperation } from "../../../redux/thunks/dialog.ts";
import { openShareEditByID } from "../../../redux/thunks/share.ts";
import SessionManager from "../../../session";
import { copyToClipboard } from "../../../util/index.ts";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import { NoWrapBox, NoWrapTypography } from "../../Common/StyledComponents.tsx";
import TimeBadge from "../../Common/TimeBadge.tsx";
import { DenseDivider, SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import FileTypeIcon from "../../FileManager/Explorer/FileTypeIcon.tsx";
import Clipboard from "../../Icons/Clipboard.tsx";
import DeleteOutlined from "../../Icons/DeleteOutlined.tsx";
import Eye from "../../Icons/Eye.tsx";
import LinkEdit from "../../Icons/LinkEdit.tsx";
import LockClosedOutlined from "../../Icons/LockClosedOutlined.tsx";
import Open from "../../Icons/Open.tsx";
import { SummaryButton } from "../Tasks/TaskCard.tsx";

export interface ShareCardProps {
  share?: Share;
  onLoad?: () => void;
  loading?: boolean;
  onShareDeleted: (id: string) => void;
}

interface ActionMenuProps extends MenuProps {
  share: Share;
  onShareDeleted: (id: string) => void;
}

const ActionMenu = ({ share, onShareDeleted, onClose, ...rest }: ActionMenuProps) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const deleteShare = useCallback(() => {
    dispatch(confirmOperation(t("fileManager.deleteShareWarning"))).then(() => {
      dispatch(sendDeleteShare(share.id)).then(() => {
        onClose && onClose({}, "backdropClick");
        enqueueSnackbar({
          message: t("application:share.shareCanceled"),
          variant: "success",
          action: DefaultCloseAction,
        });
        onShareDeleted(share.id);
      });
    });
  }, [t, share.id, onClose, dispatch, enqueueSnackbar]);

  const openEdit = useCallback(() => {
    dispatch(openShareEditByID(share.id, share.password, share.source_type == FileType.file));
    onClose && onClose({}, "backdropClick");
  }, [dispatch, share, onClose]);

  const openLink = useCallback(() => {
    window.open(share.url, "_blank");
    onClose && onClose({}, "backdropClick");
  }, [share, onClose]);

  const copyLink = useCallback(() => {
    copyToClipboard(share.url);
    onClose && onClose({}, "backdropClick");
  }, [share, onClose, t]);

  return (
    <Menu
      onClose={onClose}
      slotProps={{
        paper: {
          sx: {
            minWidth: 150,
          },
        },
      }}
      {...rest}
    >
      <SquareMenuItem dense onClick={openLink}>
        <ListItemIcon>
          <Open fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t(`fileManager.open`)}</ListItemText>
      </SquareMenuItem>
      <SquareMenuItem dense onClick={copyLink}>
        <ListItemIcon>
          <Clipboard fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t(`share.copyLinkToClipboard`)}</ListItemText>
      </SquareMenuItem>
      {!share.expired && (
        <SquareMenuItem dense onClick={openEdit}>
          <ListItemIcon>
            <LinkEdit fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t(`fileManager.${share?.expired ? "editAndReactivate" : "edit"}`)}</ListItemText>
        </SquareMenuItem>
      )}
      <DenseDivider />
      <SquareMenuItem hoverColor={theme.palette.error.light} dense onClick={deleteShare}>
        <ListItemIcon>
          <DeleteOutlined fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t(`fileManager.delete`)}</ListItemText>
      </SquareMenuItem>
    </Menu>
  );
};

const ShareCard = ({ share, onShareDeleted, onLoad, loading }: ShareCardProps) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { ref, inView } = useInView({
    rootMargin: "200px 0px",
    triggerOnce: true,
    skip: !loading || !onLoad,
  });

  const popupState = usePopupState({
    popupId: "shareAction",
    variant: "popover",
  });

  useEffect(() => {
    if (!inView) {
      return;
    }

    if (onLoad) {
      onLoad();
    }
  }, [inView]);

  const user = SessionManager.currentLoginOrNull();

  return (
    <>
      {share && <ActionMenu onShareDeleted={onShareDeleted} share={share} {...bindMenu(popupState)} />}
      <Grid item xs={12} sm={6} md={4} ref={ref}>
        <SummaryButton
          disabled={loading}
          expanded={false}
          onContextMenu={(e) => {
            e.preventDefault();
            if (share?.owner?.id === user?.user.id) {
              popupState.open(e);
            }
          }}
          sx={{ p: 0, minHeight: 0, width: "100%", textAlign: "left" }}
          {...(share?.owner?.id != user?.user.id
            ? {
                onClick: () => {
                  window.open(share?.url ?? "#", "_blank");
                },
              }
            : bindTrigger(popupState))}
        >
          <Box
            sx={{
              display: "flex",
              width: "100%",
            }}
          >
            <Box
              sx={{
                height: 48,
                p: 1.5,
              }}
            >
              {share ? (
                <FileTypeIcon
                  sx={{
                    fontSize: 32,
                  }}
                  name={share?.name ?? ""}
                  fileType={share?.source_type ?? FileType.file}
                />
              ) : (
                <Skeleton variant={"circular"} width={32} height={32} />
              )}
            </Box>
            <NoWrapBox sx={{ flexGrow: 1, p: 1, pl: 0, pr: 2 }}>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                }}
              >
                <NoWrapBox sx={{ display: "flex", gap: 0.5, alignItems: "center" }}>
                  {share && share?.password_protected && <LockClosedOutlined sx={{ fontSize: "16px" }} />}
                  <Tooltip title={share?.name ?? ""}>
                    <Typography variant={"body2"} fontWeight={500} noWrap>
                      {loading ? <Skeleton variant={"text"} width={200} /> : share?.name}
                    </Typography>
                  </Tooltip>
                </NoWrapBox>
                {share?.expired && (
                  <Chip size="small" label={t("application:share.expired")} sx={{ ml: 1, height: 18 }} />
                )}
              </Box>
              <Box>
                <Tooltip title={share?.name ?? ""}>
                  <NoWrapTypography variant={"body2"} color={"text.secondary"}>
                    {!share?.created_at ? (
                      <Skeleton variant={"text"} width={"50%"} />
                    ) : (
                      <Box
                        sx={{
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "space-between",
                        }}
                      >
                        <TimeBadge
                          sx={{
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                            overflow: "hidden",
                          }}
                          variant={"body2"}
                          datetime={share?.created_at}
                        />
                        <Box
                          sx={{
                            display: "flex",
                            alignItems: "center",
                          }}
                        >
                          <Eye sx={{ ml: 1, mr: 0.5 }} fontSize={"small"} />
                          {share?.visited ?? 0}
                        </Box>
                      </Box>
                    )}
                  </NoWrapTypography>
                </Tooltip>
              </Box>
            </NoWrapBox>
          </Box>
        </SummaryButton>
      </Grid>
    </>
  );
};

export default ShareCard;
