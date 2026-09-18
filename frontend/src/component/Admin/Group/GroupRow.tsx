import { Box, IconButton, Link, Skeleton, TableRow, Tooltip } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { deleteGroup, getGroupDetail } from "../../../api/api";
import { GroupEnt } from "../../../api/dashboard";
import { GroupPermission } from "../../../api/user";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { sizeToString } from "../../../util";
import Boolset from "../../../util/boolset";
import { NoWrapBox, NoWrapTableCell, NoWrapTypography, SquareChip } from "../../Common/StyledComponents";
import Delete from "../../Icons/Delete";
import InPrivate from "../../Icons/InPrivate";
import PersonPasskey from "../../Icons/PersonPasskey";
import Shield from "../../Icons/Shield";

export interface GroupRowProps {
  group?: GroupEnt;
  loading?: boolean;
  onDelete?: () => void;
}

export const AnonymousGroupID = 3;

const GroupRow = ({ group, loading, onDelete }: GroupRowProps) => {
  const navigate = useNavigate();
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [userCount, setUserCount] = useState<number | undefined>(undefined);
  const [countLoading, setCountLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const onPolicyClick =
    (policyId: number): ((e: React.MouseEvent<HTMLDivElement>) => void) =>
    (e) => {
      e.stopPropagation();
      navigate(`/admin/policy/${policyId}`);
    };

  const onRowClick = () => {
    navigate(`/admin/group/${group?.id}`);
  };

  const groupBs = useMemo(() => {
    return new Boolset(group?.permissions);
  }, [group?.permissions]);

  const onCountClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.stopPropagation();
    e.preventDefault();
    setCountLoading(true);
    dispatch(getGroupDetail(group?.id ?? 0, true))
      .then((res) => {
        setUserCount(res.total_users);
        setCountLoading(false);
      })
      .finally(() => {
        setCountLoading(false);
      });
  };

  const onDeleteClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    dispatch(confirmOperation(t("group.confirmDelete", { group: group?.name }))).then(() => {
      if (group?.id) {
        setDeleteLoading(true);
        dispatch(deleteGroup(group.id))
          .then(() => {
            onDelete?.();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      }
    });
  };

  const stopPropagation = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.stopPropagation();
  };

  if (loading) {
    return (
      <TableRow>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={150} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={150} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={150} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={150} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={100} />
        </NoWrapTableCell>
      </TableRow>
    );
  }
  return (
    <TableRow hover key={group?.id} sx={{ cursor: "pointer" }} onClick={onRowClick}>
      <NoWrapTableCell>{group?.id}</NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapBox sx={{ display: "flex", alignItems: "center", gap: 1 }}>
          <NoWrapTypography variant="inherit">{group?.name}</NoWrapTypography>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            {(group?.id ?? 0) <= 3 && (
              <Tooltip title={t("group.sysGroup")}>
                <Box sx={{ display: "flex", alignItems: "center" }}>
                  <Shield fontSize="small" sx={{ color: "text.secondary" }} />
                </Box>
              </Tooltip>
            )}
            {(group?.id ?? 0) == AnonymousGroupID && (
              <Tooltip title={t("group.anonymous")}>
                <Box sx={{ display: "flex", alignItems: "center" }}>
                  <InPrivate fontSize="small" sx={{ color: "text.secondary" }} />
                </Box>
              </Tooltip>
            )}
            {groupBs.enabled(GroupPermission.is_admin) && (
              <Tooltip title={t("group.adminGroup")}>
                <Box sx={{ display: "flex", alignItems: "center" }}>
                  <PersonPasskey fontSize="small" sx={{ color: "primary.main" }} />
                </Box>
              </Tooltip>
            )}
          </Box>
        </NoWrapBox>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
          {group?.edges.storage_policies && (
            <SquareChip
              onClick={onPolicyClick(group.edges.storage_policies.id)}
              key={group.edges.storage_policies.id}
              label={group.edges.storage_policies.name}
              size="small"
            />
          )}
        </Box>
      </NoWrapTableCell>
      <NoWrapTableCell>
        {countLoading ? (
          <Skeleton variant="text" width={50} />
        ) : userCount != undefined ? (
          <Link
            component={RouterLink}
            to={`/admin/user?group=${group?.id}`}
            underline="hover"
            onClick={stopPropagation}
          >
            {userCount}
          </Link>
        ) : (
          <Link href="/#" underline="hover" onClick={onCountClick}>
            {t("group.countUser")}
          </Link>
        )}
      </NoWrapTableCell>
      <NoWrapTableCell>{sizeToString(group?.max_storage ?? 0)}</NoWrapTableCell>
      <NoWrapTableCell>
        <IconButton size="small" onClick={onDeleteClick} disabled={deleteLoading || (group?.id ?? 0) <= 3}>
          <Delete fontSize="small" />
        </IconButton>
      </NoWrapTableCell>
    </TableRow>
  );
};

export default GroupRow;
