import { Icon } from "@iconify/react";
import { Avatar, Box, Divider, IconButton, Skeleton, Typography } from "@mui/material";
import Grid from "@mui/material/Grid2";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { deleteOAuthClient } from "../../../api/api";
import { GetOAuthClientResponse } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { NoWrapBox, SquareChip } from "../../Common/StyledComponents";
import CheckCircleFilled from "../../Icons/CheckCircleFilled";
import Delete from "../../Icons/Delete";
import DismissCircleFilled from "../../Icons/DismissCircleFilled";
import Info from "../../Icons/Info";
import ShieldLockFilled from "../../Icons/ShieldLockFilled";
import { BorderedCardClickable } from "../Common/AdminCard";

export interface OAuthClientCardProps {
  client?: GetOAuthClientResponse;
  onRefresh?: () => void;
  loading?: boolean;
}

const isIconUrl = (str: string) =>
  str.startsWith("http://") || str.startsWith("https://") || str.startsWith("data:") || str.startsWith("/");

const OAuthClientCard = ({ client, onRefresh, loading }: OAuthClientCardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [deleteLoading, setDeleteLoading] = useState(false);

  const handleDelete = useCallback(
    (e: React.MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      dispatch(confirmOperation(t("oauth.deleteClientConfirmation", { name: client?.name ?? "" }))).then(() => {
        setDeleteLoading(true);
        dispatch(deleteOAuthClient(client?.id ?? 0))
          .then(() => {
            onRefresh?.();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      });
    },
    [client, dispatch, onRefresh, t],
  );

  const handleEdit = useCallback(() => {
    navigate(`/admin/oauth/${client?.id}`);
  }, [client, navigate]);

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
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Skeleton variant="circular" width={32} height={32} />
              <Typography variant="subtitle1" fontWeight={600}>
                <Skeleton variant="text" width={100} />
              </Typography>
            </Box>
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

  const scopes = client?.scopes ?? [];

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
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            {client?.props?.icon ? (
              isIconUrl(client.props.icon) ? (
                <Box component="img" src={client.props.icon} sx={{ width: 32, height: 32, borderRadius: 1 }} />
              ) : (
                <Avatar sx={{ width: 32, height: 32, bgcolor: "primary.main" }}>
                  <Icon icon={client.props.icon} style={{ fontSize: 18 }} />
                </Avatar>
              )
            ) : (
              <Avatar sx={{ width: 32, height: 32, bgcolor: "primary.main" }}>
                <ShieldLockFilled sx={{ fontSize: 18 }} />
              </Avatar>
            )}
            <Typography variant="subtitle1" fontWeight={600}>
              {t(client?.name ?? "")}
            </Typography>
          </Box>
          {client?.is_system && <SquareChip label={t("oauth.systemClient")} size="small" color="primary" />}
        </Box>
        <NoWrapBox sx={{ mt: 1, mb: 2 }}>
          {scopes.length > 0 ? (
            scopes.slice(0, 3).map((scope) => <SquareChip sx={{ mr: 1 }} key={scope} label={scope} size="small" />)
          ) : (
            <Box sx={{ display: "flex", alignItems: "center", minHeight: "25px" }} color={"text.secondary"}>
              <Info sx={{ mr: 0.5, fontSize: 20 }} />
              <Typography variant="caption">{t("oauth.noScopes")}</Typography>
            </Box>
          )}
          {scopes.length > 3 && <SquareChip sx={{ mr: 1 }} label={`+${scopes.length - 3}`} size="small" />}
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
            {client?.is_enabled ? (
              <>
                <CheckCircleFilled sx={{ mr: 0.5, fontSize: 20, color: "success.main" }} />
                <Typography variant="body2" color="success.main">
                  {t("oauth.enabled")}
                </Typography>
              </>
            ) : (
              <>
                <DismissCircleFilled sx={{ mr: 0.5, fontSize: 20, color: "text.secondary" }} />
                <Typography variant="body2" color="text.secondary">
                  {t("oauth.disabled")}
                </Typography>
              </>
            )}
            {client?.total_grants !== undefined && client.total_grants > 0 && (
              <Typography variant="body2" color="text.secondary" sx={{ ml: 2 }}>
                {t("oauth.totalGrants", { count: client.total_grants })}
              </Typography>
            )}
          </Box>

          <Box>
            <IconButton size="small" onClick={handleDelete} disabled={deleteLoading || client?.is_system}>
              <Delete fontSize="small" />
            </IconButton>
          </Box>
        </Box>
      </BorderedCardClickable>
    </Grid>
  );
};

export default OAuthClientCard;
