import {
  Badge,
  Box,
  Button,
  createTheme,
  DialogContent,
  Divider,
  Grid,
  Link,
  Paper,
  Stack,
  TextField,
  ThemeProvider,
  Typography,
  useTheme,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { applyThemeWithOverrides } from "../../../../App";
import CircularProgress from "../../../Common/CircularProgress";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar";
import DraggableDialog from "../../../Dialogs/DraggableDialog";
import SideNavItem from "../../../Frame/NavBar/SideNavItem";
import Setting from "../../../Icons/Setting";

const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor"));

export interface ThemeOptionEditDialogProps {
  open: boolean;
  onClose: () => void;
  id: string;
  config: string;
  onSave: (id: string, newId: string, config: string) => void;
}

const ThemeOptionEditDialog = ({ open, onClose, id, config, onSave }: ThemeOptionEditDialogProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const { enqueueSnackbar } = useSnackbar();
  const [editedConfig, setEditedConfig] = useState(config);
  const [parsedConfig, setParsedConfig] = useState<any>(null);

  useEffect(() => {
    try {
      setParsedConfig(JSON.parse(config));
    } catch (e) {
      setParsedConfig(null);
    }
  }, [config]);

  useEffect(() => {
    try {
      setParsedConfig(JSON.parse(editedConfig));
    } catch (e) {
      // Don't update parsedConfig if JSON is invalid
    }
  }, [editedConfig]);

  const handleSave = useCallback(() => {
    try {
      // Validate JSON
      const parsed = JSON.parse(editedConfig);
      // make sure minimum config is provided
      if (!parsed.light?.palette?.primary?.main) {
        throw new Error("Invalid theme config");
      }
      // Get the new primary color (ID)
      const newId = parsed.light.palette.primary.main;
      onSave(id, newId, editedConfig);
    } catch (e) {
      enqueueSnackbar({
        message: t("settings.invalidThemeConfig"),
        variant: "warning",
        action: DefaultCloseAction,
      });
    }
  }, [editedConfig, id, onSave, enqueueSnackbar, t]);

  // Create preview themes
  const lightTheme = useMemo(() => {
    if (!parsedConfig) return null;
    try {
      return createTheme({
        palette: {
          mode: "light",
          ...parsedConfig.light.palette,
        },
      });
    } catch (e) {
      return null;
    }
  }, [parsedConfig]);

  const darkTheme = useMemo(() => {
    if (!parsedConfig) return null;
    try {
      return createTheme({
        palette: {
          mode: "dark",
          ...parsedConfig.dark.palette,
        },
      });
    } catch (e) {
      return null;
    }
  }, [parsedConfig]);

  return (
    <DraggableDialog
      title={t("settings.editThemeOption")}
      showActions
      showCancel
      onAccept={handleSave}
      dialogProps={{
        fullWidth: true,
        maxWidth: "lg",
        open,
        onClose,
      }}
    >
      <DialogContent>
        <Grid container spacing={2}>
          <Grid item xs={12} md={6}>
            <Typography variant="subtitle1" gutterBottom>
              {t("settings.themeConfiguration")}
            </Typography>
            <Suspense fallback={<CircularProgress />}>
              <MonacoEditor
                theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                value={editedConfig}
                height={"500px"}
                minHeight={"500px"}
                language={"json"}
                onChange={(e) => setEditedConfig(e as string)}
              />
            </Suspense>
            <Typography sx={{ mt: 1 }} variant="caption" color="text.secondary" gutterBottom>
              <Trans
                i18nKey={"settings.themeDes"}
                ns={"dashboard"}
                components={[
                  <Link href={"https://mui.com/material-ui/customization/default-theme/"} target={"_blank"} />,
                ]}
              />
            </Typography>
          </Grid>
          <Grid item xs={12} md={6}>
            <Typography variant="subtitle1" gutterBottom>
              {t("settings.themePreview")}
            </Typography>
            <Box sx={{ mb: 2 }}>
              <Typography variant="subtitle2" gutterBottom>
                {t("settings.lightTheme")}
              </Typography>
              {lightTheme ? (
                <ThemeProvider theme={applyThemeWithOverrides(lightTheme)}>
                  <Paper elevation={2} sx={{ p: 2, mb: 2 }}>
                    <Stack spacing={2}>
                      <Typography variant="h6">{t("settings.previewTitle")}</Typography>
                      <SideNavItem
                        active
                        label={t("settings.previewTitle")}
                        icon={<Setting fontSize="small" color="action" />}
                      />
                      <TextField label={t("settings.previewTextField")} variant="outlined" size="small" />
                      <Box>
                        <Badge badgeContent={10} color="secondary">
                          <Button variant="contained" color="primary">
                            {t("settings.previewPrimary")}
                          </Button>
                        </Badge>
                      </Box>
                    </Stack>
                  </Paper>
                </ThemeProvider>
              ) : (
                <Typography color="error">{t("settings.invalidThemePreview")}</Typography>
              )}
            </Box>

            <Divider sx={{ my: 2 }} />

            <Box>
              <Typography variant="subtitle2" gutterBottom>
                {t("settings.darkTheme")}
              </Typography>
              {darkTheme ? (
                <ThemeProvider theme={applyThemeWithOverrides(darkTheme)}>
                  <Paper elevation={2} sx={{ p: 2, mb: 2 }}>
                    <Stack spacing={2}>
                      <Typography variant="h6">{t("settings.previewTitle")}</Typography>
                      <SideNavItem
                        active
                        label={t("settings.previewTitle")}
                        icon={<Setting fontSize="small" color="action" />}
                      />
                      <TextField label={t("settings.previewTextField")} variant="outlined" size="small" />
                      <Box>
                        <Badge badgeContent={10} color="secondary">
                          <Button variant="contained" color="primary">
                            {t("settings.previewPrimary")}
                          </Button>
                        </Badge>
                      </Box>
                    </Stack>
                  </Paper>
                </ThemeProvider>
              ) : (
                <Typography color="error">{t("settings.invalidThemePreview")}</Typography>
              )}
            </Box>
          </Grid>
        </Grid>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ThemeOptionEditDialog;
