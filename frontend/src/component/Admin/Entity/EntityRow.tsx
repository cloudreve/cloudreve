import { Box, Checkbox, IconButton, Link, Skeleton, TableCell, TableRow, Tooltip } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { getEntityUrl } from "../../../api/api";
import { Entity } from "../../../api/dashboard";
import { EntityType } from "../../../api/explorer";
import { useAppDispatch } from "../../../redux/hooks";
import { sizeToString } from "../../../util";
import { NoWrapTableCell, NoWrapTypography, SquareChip } from "../../Common/StyledComponents";
import TimeBadge from "../../Common/TimeBadge";
import UserAvatar from "../../Common/User/UserAvatar";
import { cipherDisplayName } from "../../FileManager/Sidebar/BasicInfo";
import { EntityTypeText } from "../../FileManager/Sidebar/Data";
import Delete from "../../Icons/Delete";
import Download from "../../Icons/Download";
import ShieldLock from "../../Icons/ShieldLock";

export interface EntityRowProps {
  entity?: Entity;
  loading?: boolean;
  selected?: boolean;
  onDelete?: (id: number) => void;
  onSelect?: (id: number) => void;
  openEntityDialog?: (id: number) => void;
  openUserDialog?: (id: number) => void;
}

const EntityRow = ({
  entity,
  loading,
  selected,
  onDelete,
  onSelect,
  openUserDialog,
  openEntityDialog,
}: EntityRowProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [openLoading, setOpenLoading] = useState(false);

  const onSelectClick = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    onSelect?.(entity?.id ?? 0);
  };

  const onOpenClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    setOpenLoading(true);

    dispatch(getEntityUrl(entity?.id ?? 0))
      .then((url) => {
        // 直接下载文件：使用a标签的download属性强制下载
        const link = document.createElement("a");
        link.href = url;
        link.download = `entity-${entity?.id}`;
        link.style.display = "none";

        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      })
      .finally(() => {
        setOpenLoading(false);
      })
      .catch((error) => {
        console.error("Failed to get entity URL:", error);
      });
  };

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    openUserDialog?.(entity?.edges?.user?.id ?? 0);
  };

  const onDeleteClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    onDelete?.(entity?.id ?? 0);
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
          <Skeleton variant="text" width={80} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={200} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={100} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={30} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={100} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Skeleton variant="circular" width={24} height={24} />
            <Skeleton variant="text" width={100} />
          </Box>
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="circular" width={24} height={24} />
        </NoWrapTableCell>
      </TableRow>
    );
  }

  return (
    <TableRow
      hover
      key={entity?.id}
      sx={{ cursor: "pointer" }}
      onClick={() => openEntityDialog?.(entity?.id ?? 0)}
      selected={selected}
    >
      <TableCell padding="checkbox">
        <Checkbox size="small" disableRipple color="primary" onClick={onSelectClick} checked={selected} />
      </TableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{entity?.id}</NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{t(EntityTypeText[entity?.type ?? EntityType.version])}</NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
          <Tooltip title={entity?.source || ""} disableInteractive>
            <NoWrapTypography variant="inherit">{entity?.source || "-"}</NoWrapTypography>
          </Tooltip>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            {!entity?.reference_count && <SquareChip size="small" label={t("entity.waitForRecycle")} />}
            {entity?.props?.encrypt_metadata?.algorithm && (
              <Tooltip
                title={t("application:fileManager.fullEncryption", {
                  cipher: cipherDisplayName(entity?.props?.encrypt_metadata?.algorithm),
                })}
              >
                <ShieldLock
                  sx={{
                    width: "20px",
                    height: "20px",
                    color: (theme) => theme.palette.success.main,
                  }}
                />
              </Tooltip>
            )}
          </Box>
        </Box>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{sizeToString(entity?.size ?? 0)}</NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">
          <Link
            sx={{
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
            component={RouterLink}
            underline="hover"
            to={`/admin/policy/${entity?.edges?.storage_policy?.id}`}
          >
            {entity?.edges?.storage_policy?.name || "-"}
          </Link>
        </NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{entity?.reference_count ?? 0}</NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">
          <TimeBadge datetime={entity?.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
        </NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
          <UserAvatar
            sx={{ width: 24, height: 24 }}
            overwriteTextSize
            user={{
              id: entity?.user_hash_id ?? "",
              nickname: entity?.edges?.user?.nick ?? "",
              created_at: entity?.edges?.user?.created_at ?? "",
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
              {entity?.edges?.user?.nick || "-"}
            </Link>
          </NoWrapTypography>
        </Box>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
          <IconButton size="small" onClick={onOpenClick} disabled={openLoading}>
            <Download fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={onDeleteClick} disabled={deleteLoading}>
            <Delete fontSize="small" />
          </IconButton>
        </Box>
      </NoWrapTableCell>
    </TableRow>
  );
};

export default EntityRow;
