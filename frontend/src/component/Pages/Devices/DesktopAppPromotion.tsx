import { alpha, Box, Grid, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../redux/hooks.ts";

const DesktopAppPromotion = () => {
  const title = useAppSelector((state) => state.siteConfig.basic.config.title);
  const { t, i18n } = useTranslation();
  const theme = useTheme();
  const isMd = useMediaQuery(theme.breakpoints.up("md"), {
    defaultMatches: true,
  });

  return (
    <Grid
      container
      sx={{
        px: 2,
        pt: 4,
        pb: 4,
      }}
    >
      <Grid item container alignItems={"center"} xs={12} md={6}>
        <Box data-aos={isMd ? "fade-right" : "fade-up"}>
          <Box marginBottom={2}>
            <Typography
              variant="h4"
              sx={{
                fontWeight: 700,
              }}
              color="text.primary"
            >
              {t("setting.desktopAppTitleLead")}
              <Typography
                color={"primary"}
                component={"span"}
                variant={"inherit"}
                sx={{
                  background: `linear-gradient(180deg, transparent 82%, ${alpha(
                    theme.palette.secondary.main,
                    0.3,
                  )} 0%)`,
                }}
              >
                {title}
              </Typography>
              {t("setting.desktopAppTitleTrail")}
            </Typography>
          </Box>
          <Box marginBottom={3}>
            <Typography variant="h6" component="p" color="text.secondary">
              {t("setting.desktopAppDescription")}
            </Typography>
          </Box>
          <Box
            display="flex"
            flexDirection={{ xs: "column", sm: "row" }}
            alignItems={{ xs: "flex-start", sm: "center" }}
            gap={2}
            marginTop={1}
          >
            <Box
              component={"a"}
              href={"https://apps.microsoft.com/detail/9P3GH5RNNZFD?referrer=appbadge&mode=direct"}
              target={"_blank"}
              sx={{
                display: "flex",
                alignItems: "center",
                textDecoration: "none",
              }}
            >
              <Box
                component={"img"}
                src={`https://get.microsoft.com/images/${i18n.language.toLowerCase()}%20dark.svg`}
                alt="Get it from Microsoft Store"
                sx={{
                  width: 200,
                  borderRadius: 1,
                }}
              />
            </Box>
          </Box>
        </Box>
      </Grid>
      <Grid
        item
        container
        alignItems={"center"}
        justifyContent={"center"}
        xs={12}
        md={6}
        data-aos="fade-up"
        data-aos-delay="100"
      >
        <Box
          sx={{
            position: "relative",
            width: "100%",
            display: "flex",
            justifyContent: "center",
            alignItems: "flex-end",
            minHeight: { xs: 300, md: 400 },
          }}
        >
          {/* Windows Explorer Screenshot - Main larger image */}
          <Box
            sx={{
              position: "relative",
              zIndex: 1,
              border: `1px solid ${theme.palette.divider}`,
              backgroundColor: theme.palette.background.paper,
              borderRadius: "8px",
              boxShadow: theme.shadows[10],
              overflow: "hidden",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Box
              component={"img"}
              src={"https://cloudreve.org/imgs/desktop/explorer.png"}
              alt="Windows Explorer Integration"
              sx={{
                width: "100%",
                height: "100%",
                objectFit: "cover",
                filter: theme.palette.mode === "dark" ? "brightness(0.8)" : "none",
              }}
            />
          </Box>
          {/* Sync UI Screenshot - Smaller overlay image */}
          <Box
            sx={{
              position: "absolute",
              border: `1px solid ${theme.palette.divider}`,
              right: { xs: 0, md: -23 },
              bottom: { xs: -20, md: -40 },
              width: { xs: "40%", md: "40%" },
              zIndex: 2,
              backgroundColor: theme.palette.background.paper,
              borderRadius: 1,
              boxShadow: theme.shadows[15],
              overflow: "hidden",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Box
              component={"img"}
              src={"https://cloudreve.org/imgs/desktop/sync-ui.png"}
              alt="Sync UI"
              sx={{
                width: "100%",
                height: "100%",
                objectFit: "cover",
                filter: theme.palette.mode === "dark" ? "brightness(0.8)" : "none",
              }}
            />
          </Box>
        </Box>
      </Grid>
    </Grid>
  );
};

export default DesktopAppPromotion;
