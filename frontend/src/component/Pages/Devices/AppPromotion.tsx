import { alpha, Box, Grid, Typography, useTheme } from "@mui/material";
import { QRCodeSVG } from "qrcode.react";
import { useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { useAppSelector } from "../../../redux/hooks.ts";
import SessionManager from "../../../session";
import { SecondaryButton } from "../../Common/StyledComponents.tsx";

const AppPromotion = () => {
  const title = useAppSelector((state) => state.siteConfig.basic.config.title);
  const { t } = useTranslation();
  const [showQr, setShowQr] = useState(false);
  const theme = useTheme();
  const refreshToken = useMemo(() => {
    return (
      window.location.origin +
      "/login?refresh_token=" +
      (SessionManager.currentLoginOrNull()?.token?.refresh_token ?? "")
    );
  }, [showQr]);

  return (
    <Grid
      container
      sx={{
        px: 2,
        pt: 4,
        pb: 4,
      }}
    >
      <Grid item container alignItems={"center"} xs={12} md={7}>
        <Box>
          <Box marginBottom={2}>
            <Typography
              variant="h4"
              sx={{
                fontWeight: 700,
              }}
              color="text.primary"
            >
              <Trans
                i18nKey={"setting.connectByiOS"}
                values={{ title }}
                components={[
                  <Typography
                    key={0}
                    color={"primary"}
                    component={"span"}
                    variant={"inherit"}
                    sx={{
                      background: (theme) =>
                        `linear-gradient(180deg, transparent 82%, ${alpha(theme.palette.secondary.main, 0.3)} 0%)`,
                    }}
                  />,
                ]}
              />
            </Typography>
          </Box>
          <Box
            color="text.secondary"
            sx={{
              ol: {
                paddingInlineStart: "24px",
              },
              "li::marker": {
                fontSize: "1.25rem",
              },
              li: {
                marginBottom: 2,
              },
            }}
          >
            <ol>
              <li>
                <Typography variant="h6" component="p">
                  {t("setting.downloadOurApp")}
                </Typography>
                <Box marginTop={1}>
                  <Box component={"a"} href={"https://apps.apple.com/us/app/cloudreve/id1619480823"} target={"_blank"}>
                    <Box component={"img"} src={"/static/img/appstore.svg"} />
                  </Box>
                </Box>
              </li>
              <li>
                <Typography variant="h6" component="p">
                  {t("setting.fillInEndpoint")}
                </Typography>
                <Box
                  marginTop={1}
                  sx={{
                    position: "relative",
                    display: "inline-block",
                  }}
                >
                  <QRCodeSVG
                    marginSize={4}
                    size={200}
                    onClick={() => setShowQr(false)}
                    style={{
                      transition: "all .3s ease-in",
                      filter: showQr ? "initial" : "blur(8px)",
                      backgroundColor: "#fff",
                    }}
                    value={refreshToken}
                  />

                  {!showQr && (
                    <SecondaryButton
                      onClick={() => setShowQr(true)}
                      sx={{
                        position: "absolute",
                        top: "50%",
                        left: "50%",
                        transform: "translate(-50%, -50%)",
                      }}
                      color={"inherit"}
                      variant={"outlined"}
                    >
                      {t("setting.shoeQr")}
                    </SecondaryButton>
                  )}
                </Box>
              </li>
              <li>
                <Typography variant="h6" component="p">
                  {t("setting.loginApp")}
                </Typography>
              </li>
            </ol>
          </Box>
        </Box>
      </Grid>
      <Grid item xs={12} md={5}>
        <Box
          sx={{
            transform: "scale(1.5) translateY(15%)",
          }}
        >
          <Box
            sx={{
              marginX: "auto",
            }}
          >
            <Box>
              <Box>
                <Box
                  component={"img"}
                  src={"https://cloudreve.org/imgs/ios/766shots_so.png"}
                  alt="Image Description"
                  effect="blur"
                  width={1}
                  height={1}
                  sx={{
                    objectFit: "cover",
                  }}
                />
              </Box>
            </Box>
          </Box>
        </Box>
      </Grid>
    </Grid>
  );
};

export default AppPromotion;
