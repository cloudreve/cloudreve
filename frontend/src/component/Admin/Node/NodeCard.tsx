import { Box, Divider, IconButton, Skeleton, Typography } from "@mui/material";
import Grid from "@mui/material/Grid2";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { deleteNode } from "../../../api/api";
import { Node, NodeStatus, NodeType } from "../../../api/dashboard";
import { NodeCapability } from "../../../api/workflow";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import Boolset from "../../../util/boolset";
import { NoWrapBox, SquareChip } from "../../Common/StyledComponents";
import CheckCircleFilled from "../../Icons/CheckCircleFilled";
import Delete from "../../Icons/Delete";
import DismissCircleFilled from "../../Icons/DismissCircleFilled";
import Info from "../../Icons/Info";
import StarFilled from "../../Icons/StarFilled";
import { BorderedCardClickable } from "../Common/AdminCard";

export interface NodeCardProps {
  node?: Node;
  onRefresh?: () => void;
  loading?: boolean;
}

const NodeCard = ({ node, onRefresh, loading }: NodeCardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [deleteLoading, setDeleteLoading] = useState(false);

  const handleDelete = useCallback(
    (e: React.MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      dispatch(confirmOperation(t("node.deleteNodeConfirmation", { name: node?.name ?? "" }))).then(() => {
        setDeleteLoading(true);
        dispatch(deleteNode(node?.id ?? 0))
          .then(() => {
            onRefresh?.();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      });
    },
    [node, dispatch, onRefresh],
  );

  const handleEdit = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      e.stopPropagation();
      navigate(`/admin/node/${node?.id}`);
    },
    [node, navigate],
  );

  // Decode node capabilities
  const getCapabilities = useCallback(() => {
    if (!node?.capabilities) return [];

    const boolset = new Boolset(node.capabilities);
    const capabilities = [];

    if (boolset.enabled(NodeCapability.create_archive)) {
      capabilities.push({ id: NodeCapability.create_archive, name: t("application:fileManager.createArchive") });
    }
    if (boolset.enabled(NodeCapability.extract_archive)) {
      capabilities.push({ id: NodeCapability.extract_archive, name: t("application:fileManager.extractArchive") });
    }
    if (boolset.enabled(NodeCapability.remote_download)) {
      capabilities.push({ id: NodeCapability.remote_download, name: t("application:navbar.remoteDownload") });
    }

    return capabilities;
  }, [node, t]);

  // If loading is true, render a skeleton placeholder
  if (loading) {
    return (
      <Grid
        size={{
          xs: 12,
          md: 6,
          lg: 4,
        }}
      >
        <BorderedCardClickable>
          <Box
            sx={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <Typography variant="subtitle1" fontWeight={600}>
              <Skeleton variant="text" width={100} />
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <Skeleton variant="text" width={60} />
            </Typography>
          </Box>
          <NoWrapBox sx={{ mt: 1, mb: 2 }}>
            <Box sx={{ display: "flex", alignItems: "center", minHeight: "25px" }}>
              <Skeleton width={60} height={25} sx={{ mr: 1 }} />
            </Box>
          </NoWrapBox>
          <Divider sx={{ my: 1 }} />
          <Box
            sx={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <Skeleton variant="text" width="40%" height={20} />
            <Skeleton variant="circular" width={30} height={30} />
          </Box>
        </BorderedCardClickable>
      </Grid>
    );
  }

  const capabilities = getCapabilities();

  return (
    <Grid
      size={{
        xs: 12,
        md: 6,
        lg: 4,
      }}
    >
      <BorderedCardClickable onClick={handleEdit}>
        <Box
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Typography variant="subtitle1" fontWeight={600}>
            {node?.name}
          </Typography>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            {node?.type === NodeType.master && <StarFilled sx={{ mr: 0.5, fontSize: 16, color: "primary.main" }} />}
            <Typography variant="body2" color="text.secondary">
              {node?.type === NodeType.master ? t("node.master") : t("node.slave")}
            </Typography>
          </Box>
        </Box>
        <NoWrapBox sx={{ mt: 1, mb: 2 }}>
          {capabilities.length > 0 ? (
            capabilities.map((capability) => (
              <SquareChip sx={{ mr: 1 }} key={capability.id} label={capability.name} size="small" />
            ))
          ) : (
            <Box sx={{ display: "flex", alignItems: "center", minHeight: "25px" }} color={"text.secondary"}>
              <Info sx={{ mr: 0.5, fontSize: 20 }} />
              <Typography variant="caption">{t("node.noCapabilities")}</Typography>
            </Box>
          )}
        </NoWrapBox>
        <Divider sx={{ my: 1 }} />
        <Box
          sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Box sx={{ display: "flex", alignItems: "center" }}>
            {node?.status === NodeStatus.active ? (
              <>
                <CheckCircleFilled sx={{ mr: 0.5, fontSize: 20, color: "success.main" }} />
                <Typography variant="body2" color="success.main">
                  {t("node.active")}
                </Typography>
              </>
            ) : (
              <>
                <DismissCircleFilled sx={{ mr: 0.5, fontSize: 20, color: "text.secondary" }} />
                <Typography variant="body2" color="text.secondary">
                  {t("node.suspended")}
                </Typography>
              </>
            )}
          </Box>

          <Box>
            <IconButton size="small" onClick={handleDelete} disabled={deleteLoading}>
              <Delete fontSize="small" />
            </IconButton>
          </Box>
        </Box>
      </BorderedCardClickable>
    </Grid>
  );
};

export default NodeCard;
