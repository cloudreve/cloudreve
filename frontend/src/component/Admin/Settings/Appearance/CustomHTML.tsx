import { LoadingButton } from "@mui/lab";
import {
  Box,
  CircularProgress,
  Container,
  FormControl,
  Grid,
  Grid2,
  Paper,
  Stack,
  Typography,
  useTheme,
} from "@mui/material";
import { Suspense, useContext } from "react";
import { useTranslation } from "react-i18next";
import { OutlineIconTextField } from "../../../Common/Form/OutlineIconTextField";
import Logo from "../../../Common/Logo";
import DrawerHeader from "../../../Frame/NavBar/DrawerHeader";
import { SideNavItemComponent } from "../../../Frame/NavBar/PageNavigation";
import StorageSummary from "../../../Frame/NavBar/StorageSummary";
import PoweredBy from "../../../Frame/PoweredBy";
import CloudDownload from "../../../Icons/CloudDownload";
import CloudDownloadOutlined from "../../../Icons/CloudDownloadOutlined";
import CubeSync from "../../../Icons/CubeSync";
import CubeSyncFilled from "../../../Icons/CubeSyncFilled";
import MailOutlined from "../../../Icons/MailOutlined";
import PhoneLaptop from "../../../Icons/PhoneLaptop";
import PhoneLaptopOutlined from "../../../Icons/PhoneLaptopOutlined";
import SettingForm from "../../../Pages/Setting/SettingForm";
import MonacoEditor from "../../../Viewers/CodeViewer/MonacoEditor";
import { SettingContext } from "../SettingWrapper";
import { NoMarginHelperText } from "../Settings";

export interface CustomHTMLProps {}

const HeadlessFooterPreview = ({ footer, bottom }: { footer?: string; bottom?: string }) => {
  const { t } = useTranslation("application");
  return (
    <Box
      sx={{
        borderRadius: 1,
        backgroundColor: (theme) =>
          theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900],
      }}
    >
      <Container maxWidth={"xs"}>
        <Grid
          container
          spacing={0}
          direction="column"
          alignItems="center"
          justifyContent="center"
          sx={{ minHeight: "100px" }}
        >
          <Box sx={{ width: "100%", py: 2 }}>
            <Paper
              sx={{
                padding: (theme) => `${theme.spacing(2)} ${theme.spacing(3)} ${theme.spacing(3)}`,
              }}
            >
              <Logo
                sx={{
                  maxWidth: "40%",
                  maxHeight: "40px",
                  mb: 2,
                }}
              />
              <div>
                <Box
                  sx={{
                    display: "block",
                  }}
                >
                  <Typography variant={"h6"}>{t("login.siginToYourAccount")}</Typography>
                  <FormControl variant="standard" margin="normal" fullWidth>
                    <OutlineIconTextField label={t("login.email")} variant={"outlined"} icon={<MailOutlined />} />
                  </FormControl>
                  <LoadingButton sx={{ mt: 2 }} fullWidth variant="contained" color="primary">
                    <span>{t("login.continue")}</span>
                  </LoadingButton>
                  {bottom && (
                    <Box sx={{ width: "100%" }}>
                      <div dangerouslySetInnerHTML={{ __html: bottom }} />
                    </Box>
                  )}
                </Box>
              </div>
            </Paper>
          </Box>
          <PoweredBy />
          {footer && (
            <Box sx={{ mb: 2, width: "100%" }}>
              <div dangerouslySetInnerHTML={{ __html: footer }} />
            </Box>
          )}
        </Grid>
      </Container>
    </Box>
  );
};

const SidebarBottomPreview = ({ bottom }: { bottom?: string }) => {
  return (
    <Box
      sx={{
        maxWidth: "300px",
        borderRadius: 1,
        backgroundColor: (theme) =>
          theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900],
      }}
    >
      <DrawerHeader disabled />
      <Stack
        direction={"column"}
        spacing={2}
        sx={{
          px: 1,
          pb: 1,
          flexGrow: 1,
          mx: 1,
          overflow: "auto",
        }}
      >
        <Box sx={{ width: "100%" }}>
          <SideNavItemComponent
            item={{
              label: "navbar.remoteDownload",
              icon: [CloudDownload, CloudDownloadOutlined],
              path: "#1",
            }}
          />
          <SideNavItemComponent
            item={{
              label: "navbar.connect",
              icon: [PhoneLaptop, PhoneLaptopOutlined],
              path: "#1",
            }}
          />
          <SideNavItemComponent
            item={{
              label: "navbar.taskQueue",
              icon: [CubeSyncFilled, CubeSync],
              path: "#1",
            }}
          />
        </Box>
        <StorageSummary />
        {bottom && (
          <Box sx={{ width: "100%" }}>
            <div dangerouslySetInnerHTML={{ __html: bottom ?? "" }} />
          </Box>
        )}
      </Stack>
    </Box>
  );
};

const CustomHTML = ({}: CustomHTMLProps) => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);
  const theme = useTheme();

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        {t("settings.customHTML")}
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        {t("settings.customHTMLDes")}
      </Typography>
      <Stack spacing={3}>
        <SettingForm
          title={t("settings.headlessFooter")}
          lgWidth={5}
          spacing={3}
          secondary={
            <Grid2 size={{ md: 7, xs: 12 }}>
              <HeadlessFooterPreview footer={values.headless_footer_html ?? ""} />
            </Grid2>
          }
        >
          <FormControl fullWidth>
            <Suspense fallback={<CircularProgress />}>
              <MonacoEditor
                theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                value={values.headless_footer_html}
                height={"300px"}
                minHeight={"300px"}
                language={"html"}
                onChange={(e) => setSettings({ headless_footer_html: e as string })}
              />
            </Suspense>
            <NoMarginHelperText>{t("settings.headlessFooterDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm
          title={t("settings.headlessBottom")}
          lgWidth={5}
          spacing={3}
          secondary={
            <Grid2 size={{ md: 7, xs: 12 }}>
              <HeadlessFooterPreview bottom={values.headless_bottom_html ?? ""} />
            </Grid2>
          }
        >
          <FormControl fullWidth>
            <Suspense fallback={<CircularProgress />}>
              <MonacoEditor
                theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                value={values.headless_bottom_html}
                height={"300px"}
                minHeight={"300px"}
                language={"html"}
                onChange={(e) => setSettings({ headless_bottom_html: e as string })}
              />
            </Suspense>
            <NoMarginHelperText>{t("settings.headlessBottomDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm
          title={t("settings.sidebarBottom")}
          lgWidth={5}
          spacing={3}
          secondary={
            <Grid2 size={{ md: 7, xs: 12 }}>
              <SidebarBottomPreview bottom={values.sidebar_bottom_html ?? ""} />
            </Grid2>
          }
        >
          <FormControl fullWidth>
            <Suspense fallback={<CircularProgress />}>
              <MonacoEditor
                theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                value={values.sidebar_bottom_html}
                height={"300px"}
                minHeight={"300px"}
                language={"html"}
                onChange={(e) => setSettings({ sidebar_bottom_html: e as string })}
              />
            </Suspense>
            <NoMarginHelperText>{t("settings.sidebarBottomDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </Stack>
    </Box>
  );
};

export default CustomHTML;
