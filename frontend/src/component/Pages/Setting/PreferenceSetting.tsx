import { LoadingButton } from "@mui/lab";
import {
  Box,
  Checkbox,
  Chip,
  Collapse,
  FormControl,
  FormHelperText,
  ListItemText,
  Stack,
  styled,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import i18next from "i18next";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendUpdateUserSetting } from "../../../api/api.ts";
import { UserSettings as UserSettingsType } from "../../../api/user.ts";
import { languages } from "../../../i18n.ts";
import { setPreferredTheme } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { clearLocalCustomView } from "../../../redux/thunks/filemanager.ts";
import { selectLanguage } from "../../../redux/thunks/settings.ts";
import SessionManager, { UserSettings } from "../../../session";
import { refreshTimeZone, timeZone } from "../../../util/datetime.ts";
import {
  DenseAutocomplete,
  DenseFilledTextField,
  DenseSelect,
  SmallFormControlLabel,
} from "../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import { ColorCircle, SelectorBox } from "../../FileManager/FileInfo/ColorCircle/CircleColorSelector.tsx";
import { SwitchPopover } from "../../Frame/NavBar/DarkThemeSwitcher.tsx";
import CheckboxChecked from "../../Icons/CheckboxChecked.tsx";
import FolderArrowRightOutlined from "../../Icons/FolderArrowRightOutlined.tsx";
import RectangleLandscapeSync from "../../Icons/RectangleLandscapeSync.tsx";
import RectangleLandscapeSyncOff from "../../Icons/RectangleLandscapeSyncOff.tsx";
import SettingForm from "./SettingForm.tsx";

export interface PreferenceSettingProps {
  setting: UserSettingsType;
  setSetting: (setting: UserSettingsType) => void;
}

declare namespace Intl {
  type Key = "calendar" | "collation" | "currency" | "numberingSystem" | "timeZone" | "unit";

  function supportedValuesOf(input: Key): string[];
}

interface ThemeOption {
  [key: string]: any;
}

const OutlinedSettingBox = styled(Box)(({ theme }) => ({
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(1),
  border: `1px solid ${theme.palette.mode === "light" ? "rgba(0, 0, 0, 0.23)" : "rgba(255, 255, 255, 0.23)"}`,
}));

const PreferenceSetting = ({ setting, setSetting }: PreferenceSettingProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const { themes } = useAppSelector((s) => s.siteConfig.basic.config);

  const [timeZoneValue, setTimeZoneValue] = useState(timeZone);
  const versionMaxRef = useRef<HTMLInputElement>(null);
  const [loading, setLoading] = useState(false);
  const preferredTheme = useAppSelector((state) => state.globalState.preferredTheme);
  const defaultTheme = useAppSelector((state) => state.siteConfig.basic.config.default_theme);
  const [versionRetentionEnabled, setVersionRetentionEnabled] = useState(setting.version_retention_enabled);
  const [versionRetentionMax, setVersionRetentionMax] = useState(setting.version_retention_max);
  const [versionRetentionExts, setVersionRetentionExts] = useState<string[] | undefined>(setting.version_retention_ext);
  const [showSaveButton, setShowSaveButton] = useState(false);
  const [autoExpandTreeView, setAutoExpandTreeView] = useState(
    SessionManager.getWithFallback(UserSettings.TreeViewAutoExpand),
  );
  const [folderClickAction, setFolderClickAction] = useState(
    SessionManager.getWithFallback(UserSettings.FolderClickAction),
  );

  const onRetentionCheckChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setVersionRetentionEnabled(e.target.checked);
    setShowSaveButton(true);
  };

  const onVersionRetentionMaxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setVersionRetentionMax(parseInt(e.target.value) ?? 0);
    setShowSaveButton(true);
  };

  const newExtAdded = (_e: any, newValue: string[]) => {
    // Remove start dots in each value if presented
    setVersionRetentionExts(newValue.map((v) => v.replace(/^\./, "")));
    setShowSaveButton(true);
  };

  useEffect(() => {
    setVersionRetentionEnabled(setting.version_retention_enabled);
    setVersionRetentionMax(setting.version_retention_max);
    setVersionRetentionExts(setting.version_retention_ext);
  }, [setting]);

  const selectTimeZone = (value: string) => {
    setTimeZoneValue(value);
    SessionManager.set(UserSettings.TimeZone, value);
    refreshTimeZone();
  };

  const themeOptions: ThemeOption = useMemo(() => {
    if (themes) {
      return JSON.parse(themes);
    }

    return {};
  }, [themes]);

  const applyTheme = (color: string) => {
    dispatch(setPreferredTheme(color));
    dispatch(
      sendUpdateUserSetting({
        preferred_theme: color,
      }),
    );
  };

  const saveVersionRetention = () => {
    setLoading(true);
    dispatch(
      sendUpdateUserSetting({
        version_retention_enabled: versionRetentionEnabled,
        version_retention_max: versionRetentionMax,
        version_retention_ext: versionRetentionExts,
      }),
    )
      .then(() => {
        setShowSaveButton(false);
        setSetting({
          ...setting,
          version_retention_enabled: versionRetentionEnabled,
          version_retention_max: versionRetentionMax,
          version_retention_ext: versionRetentionExts,
        });
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const onDisableViewSyncChange = (e: React.MouseEvent<HTMLElement>, enabled: boolean) => {
    setSetting({ ...setting, disable_view_sync: !enabled });
    setLoading(true);
    dispatch(
      sendUpdateUserSetting({
        disable_view_sync: !enabled,
      }),
    )
      .then(() => {
        const user = SessionManager.currentLoginOrNull();
        if (user?.user) {
          SessionManager.updateUserIfExist({
            ...user?.user,
            disable_view_sync: !enabled,
          });
        }
        clearLocalCustomView();
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const onAutoExpandTreeViewChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setAutoExpandTreeView(e.target.checked);
    SessionManager.set(UserSettings.TreeViewAutoExpand, e.target.checked);
  };

  const onFolderClickActionChange = (_e: React.MouseEvent<HTMLElement>, value: string | null) => {
    if (value) {
      setFolderClickAction(value);
      SessionManager.set(UserSettings.FolderClickAction, value);
    }
  };

  return (
    <Stack spacing={3}>
      <SettingForm title={t("setting.language")} lgWidth={3}>
        <FormControl fullWidth>
          <DenseSelect value={i18next.language} onChange={(e) => dispatch(selectLanguage(e.target.value as string))}>
            {languages.map((l) => (
              <SquareMenuItem value={l.code}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {l.displayName}
                </ListItemText>
              </SquareMenuItem>
            ))}
          </DenseSelect>
          <FormHelperText>{t("setting.languageDes")}</FormHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("setting.timeZone")} lgWidth={4}>
        <FormControl fullWidth>
          <DenseSelect value={timeZoneValue} onChange={(e) => selectTimeZone(e.target.value as string)}>
            {Intl.supportedValuesOf("timeZone").map((v) => (
              <SquareMenuItem value={v}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {v}
                </ListItemText>
              </SquareMenuItem>
            ))}
          </DenseSelect>
          <FormHelperText>{t("setting.timezoneDes")}</FormHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("setting.darkMode")} lgWidth={12}>
        <SwitchPopover />
      </SettingForm>
      <SettingForm title={t("setting.themeColor")} lgWidth={12}>
        <SelectorBox sx={{ gap: 1 }}>
          {Object.keys(themeOptions).map((color, index) => (
            <ColorCircle
              size={30}
              color={color}
              onClick={() => applyTheme(color)}
              selected={(preferredTheme && preferredTheme == color) || (!preferredTheme && defaultTheme == color)}
            />
          ))}
        </SelectorBox>
      </SettingForm>
      <SettingForm title={t("setting.versionRetention")}>
        <OutlinedSettingBox sx={{ py: 0.5 }}>
          <Box
            sx={{
              pl: 1,
              display: "flex",
              flexDirection: isMobile ? "column" : "row",
              alignItems: isMobile ? "flex-start" : "center",
            }}
          >
            <SmallFormControlLabel
              control={<Checkbox size="small" checked={versionRetentionEnabled} onChange={onRetentionCheckChange} />}
              label={t("application:setting.enableVersionRetention")}
            />
            <Typography variant={"caption"} color={"text.secondary"}>
              {t("application:setting.enableVersionRetentionDes")}
            </Typography>
          </Box>
          <Stack spacing={1} sx={{ mb: versionRetentionEnabled || showSaveButton ? 0.5 : 0 }}>
            <Collapse in={versionRetentionEnabled} unmountOnExit>
              <Stack
                sx={{
                  mt: 1,
                }}
                spacing={1}
              >
                <DenseAutocomplete
                  multiple
                  options={[]}
                  value={versionRetentionExts ?? []}
                  autoSelect
                  freeSolo
                  onChange={newExtAdded}
                  renderTags={(value: readonly string[], getTagProps) =>
                    value.map((option: string, index: number) => {
                      const { key, ...tagProps } = getTagProps({ index });
                      return <Chip variant="outlined" label={option} key={key} size={"small"} {...tagProps} />;
                    })
                  }
                  renderInput={(params) => (
                    <DenseFilledTextField
                      label={t("setting.versionRetentionEnabledExt")}
                      {...params}
                      sx={{
                        "& .MuiInputBase-root": {},
                        "& .MuiInputBase-root.MuiOutlinedInput-root": {
                          paddingTop: theme.spacing(0.6),
                          paddingBottom: theme.spacing(0.6),
                        },
                        mt: 0,
                      }}
                      helperText={t("application:setting.versionRetentionEnabledExtDes")}
                      margin="dense"
                      type="text"
                      fullWidth
                    />
                  )}
                />
                <Box>
                  <DenseFilledTextField
                    onChange={onVersionRetentionMaxChange}
                    value={versionRetentionMax}
                    required
                    inputProps={{
                      type: "number",
                      step: 1,
                      min: 0,
                    }}
                    helperText={t("setting.versionRetentionMax")}
                    inputRef={versionMaxRef}
                  />
                </Box>
              </Stack>
            </Collapse>
            <Collapse in={showSaveButton} unmountOnExit>
              <LoadingButton variant={"contained"} loading={loading} onClick={saveVersionRetention}>
                <span>{t("fileManager.save")}</span>
              </LoadingButton>
            </Collapse>
          </Stack>
        </OutlinedSettingBox>
      </SettingForm>
      <SettingForm title={t("setting.syncView")} lgWidth={12}>
        <ToggleButtonGroup
          color="primary"
          value={!setting.disable_view_sync}
          exclusive
          disabled={loading}
          onChange={onDisableViewSyncChange}
          size={"small"}
          aria-label="Platform"
        >
          <ToggleButton value={true}>
            <RectangleLandscapeSync fontSize="small" sx={{ mr: 1 }} />
            {t("setting.syncViewOn")}
          </ToggleButton>
          <ToggleButton value={false}>
            <RectangleLandscapeSyncOff fontSize="small" sx={{ mr: 1 }} />
            {t("setting.syncViewOff")}
          </ToggleButton>
        </ToggleButtonGroup>
        <FormHelperText>{t("setting.syncViewDes")}</FormHelperText>
      </SettingForm>
      <SettingForm title={t("setting.treeView")} lgWidth={12}>
        <SmallFormControlLabel
          control={<Checkbox size="small" checked={autoExpandTreeView} onChange={onAutoExpandTreeViewChange} />}
          label={t("application:setting.autoExpandTreeView")}
        />
        <FormHelperText>{t("setting.autoExpandTreeViewDes")}</FormHelperText>
      </SettingForm>
      <SettingForm title={t("setting.folderClickAction")} lgWidth={12}>
        <ToggleButtonGroup
          color="primary"
          value={folderClickAction}
          exclusive
          onChange={onFolderClickActionChange}
          size={"small"}
          aria-label="Folder click action"
        >
          <ToggleButton value={"open"}>
            <FolderArrowRightOutlined fontSize="small" sx={{ mr: 1 }} />
            {t("setting.folderClickOpen")}
          </ToggleButton>
          <ToggleButton value={"select"}>
            <CheckboxChecked fontSize="small" sx={{ mr: 1 }} />
            {t("setting.folderClickSelect")}
          </ToggleButton>
        </ToggleButtonGroup>
        <FormHelperText>{t("setting.folderClickActionDes")}</FormHelperText>
      </SettingForm>
    </Stack>
  );
};

export default PreferenceSetting;
