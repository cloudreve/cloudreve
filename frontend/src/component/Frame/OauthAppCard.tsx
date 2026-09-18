import { Icon } from "@iconify/react";
import { Avatar, Box, Paper, Skeleton, styled, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { AppRegistration } from "../../api/user";
import Link from "../Icons/Link";

interface OAuthAppCardProps {
  app: AppRegistration | undefined;
  loading: boolean;
}

const StyledPaper = styled(Paper)(({ theme }) => ({
  padding: theme.spacing(2),
  boxShadow: "none",
  border: `1px solid ${theme.palette.divider}`,
}));

const isUrl = (str: string) =>
  str.startsWith("http://") || str.startsWith("https://") || str.startsWith("data:") || str.startsWith("/");

const AppIcon = ({ icon }: { icon?: string }) => {
  if (!icon) {
    return (
      <Avatar
        sx={{
          width: 48,
          height: 48,
          bgcolor: (theme) => theme.palette.primary.main,
        }}
      >
        <Link />
      </Avatar>
    );
  }

  if (isUrl(icon)) {
    return (
      <Box
        component="img"
        src={icon}
        sx={{
          width: 48,
          height: 48,
          maxWidth: 48,
          maxHeight: 48,
          objectFit: "contain",
          borderRadius: 1,
        }}
      />
    );
  }

  return (
    <Avatar
      sx={{
        width: 48,
        height: 48,
        bgcolor: (theme) => theme.palette.primary.main,
      }}
    >
      <Icon icon={icon} style={{ fontSize: 24 }} />
    </Avatar>
  );
};

export const OAuthAppCard = ({ app, loading }: OAuthAppCardProps) => {
  const { t } = useTranslation();

  if (loading) {
    return (
      <StyledPaper>
        <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
          <Skeleton variant="circular" width={48} height={48} />
          <Box sx={{ flex: 1 }}>
            <Skeleton variant="text" width="60%" height={24} />
            <Skeleton variant="text" width="80%" height={20} />
          </Box>
        </Box>
      </StyledPaper>
    );
  }

  if (!app) return null;

  return (
    <StyledPaper>
      <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
        <AppIcon icon={app.icon} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="subtitle1" fontWeight="medium" noWrap>
            {t(app.name)}
          </Typography>
          <Typography variant="body2" color="text.secondary" noWrap>
            {t("oauth.wantsToConnect")}
          </Typography>
        </Box>
      </Box>
    </StyledPaper>
  );
};

export const ConnectingLine = () => {
  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      {/* Dotted line */}
      <Box
        sx={{
          width: 2,
          height: 16,
          backgroundImage: (theme) => `linear-gradient(${theme.palette.divider} 50%, transparent 50%)`,
          backgroundSize: "2px 6px",
        }}
      />
      {/* Link icon */}
      <Box
        sx={{
          width: 28,
          height: 28,
          borderRadius: "50%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          bgcolor: (theme) => theme.palette.background.paper,
          border: (theme) => `2px dashed ${theme.palette.divider}`,
        }}
      >
        <Link sx={{ fontSize: 14, color: "text.secondary" }} />
      </Box>
      {/* Dotted line */}
      <Box
        sx={{
          width: 2,
          height: 16,
          backgroundImage: (theme) => `linear-gradient(${theme.palette.divider} 50%, transparent 50%)`,
          backgroundSize: "2px 6px",
        }}
      />
    </Box>
  );
};
