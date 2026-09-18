import { Box, Checkbox, IconButton, Link, Skeleton, TableCell, TableRow } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Share } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { NoWrapTableCell, NoWrapTypography } from "../../Common/StyledComponents";
import TimeBadge from "../../Common/TimeBadge";
import UserAvatar from "../../Common/User/UserAvatar";
import FileTypeIcon from "../../FileManager/Explorer/FileTypeIcon";
import { ShareExpires } from "../../FileManager/TopBar/ShareInfoPopover";
import Delete from "../../Icons/Delete";
import Open from "../../Icons/Open";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { batchDeleteShares } from "../../../api/api";

export interface ShareRowProps {
  share?: Share;
  loading?: boolean;
  deleting?: boolean;
  selected?: boolean;
  onDelete?: () => void;
  onDetails?: (id: number) => void;
  onSelect?: (id: number) => void;
  openUserDialog?: (id: number) => void;
  openFileDialog?: (id: number) => void;
}

const ShareRow = ({
  share,
  loading,
  deleting,
  selected,
  onDelete,
  onDetails,
  onSelect,
  openUserDialog,
  openFileDialog,
}: ShareRowProps) => {
  const navigate = useNavigate();
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [openLoading, setOpenLoading] = useState(false);
  const onRowClick = () => {
    onDetails?.(share?.id ?? 0);
  };

  const onDeleteClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    dispatch(confirmOperation(t("share.confirmDelete"))).then(() => {
      if (share?.id) {
        setDeleteLoading(true);
        dispatch(batchDeleteShares({ ids: [share.id] }))
          .then(() => {
            onDelete?.();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      }
    });
  };

  const onSelectClick = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    onSelect?.(share?.id ?? 0);
  };

  if (loading) {
    return (
      <TableRow sx={{ height: "43px" }}>
        <NoWrapTableCell>
          <Skeleton variant="circular" width={24} height={24} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={30} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Skeleton variant="circular" width={24} height={24} />
            <Skeleton variant="text" width={200} />
          </Box>
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={130} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Skeleton variant="circular" width={24} height={24} />
            <Skeleton variant="text" width={50} />
          </Box>
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={70} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="circular" width={24} height={24} />
        </NoWrapTableCell>
      </TableRow>
    );
  }

  const stopPropagation = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
  };

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    openUserDialog?.(share?.edges?.user?.id ?? 0);
  };

  const fileClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    openFileDialog?.(share?.edges?.file?.id ?? 0);
  };

  const onOpenLink = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    window.open(share?.share_link ?? "", "_blank");
  };

  return (
    <TableRow hover key={share?.id} sx={{ cursor: "pointer" }} onClick={onRowClick} selected={selected}>
      <TableCell padding="checkbox">
        <Checkbox
          disabled={deleting}
          size="small"
          disableRipple
          color="primary"
          onClick={onSelectClick}
          checked={selected}
        />
      </TableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{share?.id}</NoWrapTypography>
      </NoWrapTableCell>
      <TableCell>
        {share?.edges?.file ? (
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <FileTypeIcon name={share?.edges?.file?.name ?? ""} fileType={share?.edges?.file?.type ?? 0} />
            <NoWrapTypography variant="inherit">
              <Link href={`#/`} onClick={fileClicked} underline="hover">
                {share?.edges?.file?.name}
              </Link>
            </NoWrapTypography>
          </Box>
        ) : (
          <em>{t("share.deleted")}</em>
        )}
      </TableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{share?.views ?? 0}</NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{share?.downloads ?? 0}</NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{share?.price}</NoWrapTypography>
      </NoWrapTableCell>
      <TableCell>
        <ShareExpires expires={share?.expires} remain_downloads={share?.remain_downloads} />
      </TableCell>
      <NoWrapTableCell>
        {share?.edges?.user && (
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <UserAvatar
              sx={{ width: 24, height: 24 }}
              overwriteTextSize
              user={{
                id: share?.user_hash_id ?? "",
                nickname: share?.edges?.user?.nick ?? "",
                created_at: share?.edges?.user?.created_at ?? "",
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
                {share?.edges?.user?.nick}
              </Link>
            </NoWrapTypography>
          </Box>
        )}
      </NoWrapTableCell>
      <NoWrapTableCell>
        <TimeBadge datetime={share?.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
      </NoWrapTableCell>
      <NoWrapTableCell>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
          <IconButton size="small" onClick={onOpenLink} disabled={deleteLoading || deleting}>
            <Open fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={onDeleteClick} disabled={deleteLoading || deleting}>
            <Delete fontSize="small" />
          </IconButton>
        </Box>
      </NoWrapTableCell>
    </TableRow>
  );
};

export default ShareRow;
