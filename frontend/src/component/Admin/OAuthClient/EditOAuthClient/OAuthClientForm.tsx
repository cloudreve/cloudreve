import { Icon } from "@iconify/react";
import { Description } from "@mui/icons-material";
import {
  Alert,
  alpha,
  Avatar,
  Box,
  Checkbox,
  Chip,
  Divider,
  FormControl,
  FormControlLabel,
  Grow,
  IconButton,
  InputAdornment,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Stack,
  Switch,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
  useTheme,
} from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { GetOAuthClientResponse } from "../../../../api/dashboard";
import { copyToClipboard } from "../../../../util";
import { Code } from "../../../Common/Code";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents";
import ArrowSync from "../../../Icons/ArrowSync";
import CloudOff from "../../../Icons/CloudOff";
import CopyOutlined from "../../../Icons/CopyOutlined";
import CubeSync from "../../../Icons/CubeSync";
import Currency from "../../../Icons/Currency";
import FolderOutlined from "../../../Icons/FolderOutlined";
import Link from "../../../Icons/Link";
import LockClosedOutlined from "../../../Icons/LockClosedOutlined";
import PersonOutlined from "../../../Icons/PersonOutlined";
import PhoneLaptopOutlined from "../../../Icons/PhoneLaptopOutlined";
import ShareOutlined from "../../../Icons/ShareOutlined";
import WrenchSettings from "../../../Icons/WrenchSettings";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { BorderedCard } from "../../Common/AdminCard";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { OAuthClientSettingContext } from "./OAuthClientSettingWrapper";

// Scope definitions matching the consent page
interface ScopeInfo {
  name: string;
  description: string;
  icon: React.ElementType;
  category: "profile" | "files" | "security" | "admin" | "other";
  readScope?: string;
  writeScope?: string;
}

const scopeDefinitions: Record<string, ScopeInfo> = {
  offline_access: {
    name: "oauth.scope.offlineAccess",
    description: "oauth.scope.offlineAccessDesc",
    icon: CloudOff,
    category: "other",
  },
  "UserSecurityInfo.Read": {
    name: "oauth.scope.userSecurityInfo",
    description: "oauth.scope.userSecurityInfoDesc",
    icon: LockClosedOutlined,
    category: "security",
  },
  "Workflow.Read": {
    name: "navbar.taskQueue",
    description: "oauth.scope.workflowDesc",
    icon: CubeSync,
    category: "other",
    readScope: "Workflow.Read",
    writeScope: "Workflow.Write",
  },
  "Admin.Read": {
    name: "oauth.scope.admin",
    description: "oauth.scope.adminDesc",
    icon: WrenchSettings,
    category: "admin",
    readScope: "Admin.Read",
    writeScope: "Admin.Write",
  },
  "Files.Read": {
    name: "oauth.scope.files",
    description: "oauth.scope.filesDesc",
    icon: FolderOutlined,
    category: "files",
    readScope: "Files.Read",
    writeScope: "Files.Write",
  },
  "Shares.Read": {
    name: "oauth.scope.shares",
    description: "oauth.scope.sharesDesc",
    icon: ShareOutlined,
    category: "files",
    readScope: "Shares.Read",
    writeScope: "Shares.Write",
  },
  "Finance.Read": {
    name: "setting.finance",
    description: "oauth.scope.financeDesc",
    icon: Currency,
    category: "other",
    readScope: "Finance.Read",
    writeScope: "Finance.Write",
  },
  "DavAccount.Read": {
    name: "oauth.scope.davAccount",
    description: "oauth.scope.davAccountDesc",
    icon: PhoneLaptopOutlined,
    category: "security",
    readScope: "DavAccount.Read",
    writeScope: "DavAccount.Write",
  },
  "UserInfo.Read": {
    name: "oauth.scope.profile",
    description: "oauth.scope.profileDesc",
    icon: PersonOutlined,
    category: "profile",
    readScope: "UserInfo.Read",
    writeScope: "UserInfo.Write",
  },
};

// All available scopes organized by category
const availableScopes = [
  { scope: "openid", required: true },
  { scope: "email", required: false },
  { scope: "profile", required: false },
  { scope: "offline_access", required: false },
  { scope: "UserInfo.Read", hasWrite: true },
  { scope: "UserSecurityInfo.Read", hasWrite: true },
  { scope: "Files.Read", hasWrite: true },
  { scope: "Shares.Read", hasWrite: true },
  { scope: "Workflow.Read", hasWrite: true },
  { scope: "Finance.Read", hasWrite: true },
  { scope: "DavAccount.Read", hasWrite: true },
  { scope: "Admin.Read", hasWrite: true },
];

const generateSecret = (length: number = 32): string => {
  const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return Array.from(array, (byte) => charset[byte % charset.length]).join("");
};

const isIconUrl = (str: string) =>
  str.startsWith("http://") || str.startsWith("https://") || str.startsWith("data:") || str.startsWith("/");

const IconPreview = ({ icon }: { icon?: string }) => {
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

  if (isIconUrl(icon)) {
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

const OAuthClientForm = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const { values, setClient, formRef, isNew } = useContext(OAuthClientSettingContext);
  const [showSecret, setShowSecret] = useState(isNew);

  const onNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setClient((p: GetOAuthClientResponse) => ({ ...p, name: e.target.value }));
    },
    [setClient],
  );

  const onIconChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setClient((p: GetOAuthClientResponse) => ({
        ...p,
        props: { ...p.props, icon: e.target.value },
      }));
    },
    [setClient],
  );

  const onSecretChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setClient((p: GetOAuthClientResponse) => ({ ...p, secret: e.target.value }));
    },
    [setClient],
  );

  const onRedirectUrisChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const uris = e.target.value.split("\n").filter((uri) => uri.trim() !== "");
      setClient((p: GetOAuthClientResponse) => ({ ...p, redirect_uris: uris }));
    },
    [setClient],
  );

  const onRefreshTokenTTLChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const ttl = parseInt(e.target.value);
      setClient((p: GetOAuthClientResponse) => ({
        ...p,
        props: { ...p.props, refresh_token_ttl: isNaN(ttl) ? 2592000 : ttl },
      }));
    },
    [setClient],
  );

  const onEnabledChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setClient((p: GetOAuthClientResponse) => ({ ...p, is_enabled: e.target.checked }));
    },
    [setClient],
  );

  const onGenerateSecret = useCallback(() => {
    const newSecret = generateSecret(32);
    setClient((p: GetOAuthClientResponse) => ({ ...p, secret: newSecret }));
    setShowSecret(true);
  }, [setClient]);

  const onCopyToClipboard = useCallback((text: string) => {
    copyToClipboard(text);
  }, []);

  const redirectUrisValue = useMemo(() => {
    return (values.redirect_uris || []).join("\n");
  }, [values.redirect_uris]);

  const onScopeToggle = useCallback(
    (scope: string) => {
      setClient((p: GetOAuthClientResponse) => {
        const currentScopes = new Set(p.scopes || []);
        const scopeInfo = scopeDefinitions[scope];
        const readScope = scopeInfo?.readScope || scope;
        const writeScope = scopeInfo?.writeScope || scope.replace(".Read", ".Write");

        // Check if scope exists (either .Read or .Write, since .Write implies .Read)
        const hasReadOrWrite = currentScopes.has(readScope) || currentScopes.has(writeScope);

        if (hasReadOrWrite) {
          // Removing scope removes both variants
          currentScopes.delete(readScope);
          currentScopes.delete(writeScope);
        } else {
          // Adding scope defaults to read-only
          currentScopes.add(readScope);
        }

        return { ...p, scopes: Array.from(currentScopes) };
      });
    },
    [setClient],
  );

  const onAccessLevelChange = useCallback(
    (scope: string, newLevel: "read" | "readwrite" | null) => {
      if (!newLevel) return; // Don't allow deselecting when clicking the same option

      setClient((p: GetOAuthClientResponse) => {
        const currentScopes = new Set(p.scopes || []);
        const scopeInfo = scopeDefinitions[scope];
        const readScope = scopeInfo?.readScope || scope;
        const writeScope = scopeInfo?.writeScope || scope.replace(".Read", ".Write");

        if (newLevel === "read") {
          // .Read only - remove .Write if present, add .Read
          currentScopes.delete(writeScope);
          currentScopes.add(readScope);
        } else if (newLevel === "readwrite") {
          // .Write implies .Read, so only store .Write
          currentScopes.delete(readScope);
          currentScopes.add(writeScope);
        }

        return { ...p, scopes: Array.from(currentScopes) };
      });
    },
    [setClient],
  );

  const hasScope = useCallback(
    (scope: string) => {
      const scopes = values.scopes || [];
      // For scopes with .Read/.Write suffix, check if either variant exists
      if (scope.endsWith(".Read") || scope.endsWith(".Write")) {
        const baseScope = scope.replace(/\.(Read|Write)$/, "");
        return scopes.some((s) => s === `${baseScope}.Read` || s === `${baseScope}.Write`);
      }
      return scopes.includes(scope);
    },
    [values.scopes],
  );

  const hasWriteScope = useCallback(
    (scope: string) => {
      const writeScope = scope.replace(".Read", ".Write");
      return (values.scopes || []).includes(writeScope);
    },
    [values.scopes],
  );

  const getAccessLevel = useCallback(
    (scope: string): "read" | "readwrite" | null => {
      if (!hasScope(scope)) return null;
      return hasWriteScope(scope) ? "readwrite" : "read";
    },
    [hasScope, hasWriteScope],
  );

  return (
    <Box component="form" ref={formRef} noValidate>
      <Stack spacing={5}>
        {/* Basic Information Section */}
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("policy.basicInfo")}
          </Typography>
          <SettingSectionContent>
            {values.is_system && (
              <SettingForm lgWidth={5}>
                <Alert severity="info">{t("oauth.systemClientNote")}</Alert>
              </SettingForm>
            )}

            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={<Switch checked={values.is_enabled ?? true} onChange={onEnabledChange} />}
                  label={t("oauth.enableClient")}
                />
                <NoMarginHelperText>{t("oauth.enableClientDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("oauth.clientName")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField required value={values.name || ""} onChange={onNameChange} />
                <NoMarginHelperText>
                  <Trans i18nKey="oauth.clientNameDes" ns="dashboard" components={[<Code />]} />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("oauth.clientIcon")} lgWidth={5}>
              <FormControl fullWidth>
                <Box sx={{ display: "flex", alignItems: "flex-start", gap: 2 }}>
                  <IconPreview icon={values.props?.icon} />
                  <Box sx={{ flex: 1 }}>
                    <DenseFilledTextField
                      required
                      value={values.props?.icon || ""}
                      onChange={onIconChange}
                      placeholder="mdi:application"
                      fullWidth
                    />
                    <NoMarginHelperText>{t("oauth.clientIconDes")}</NoMarginHelperText>
                  </Box>
                </Box>
              </FormControl>
            </SettingForm>

            {!isNew && values.guid && (
              <SettingForm title={t("oauth.clientId")} lgWidth={5}>
                <FormControl fullWidth>
                  <DenseFilledTextField
                    value={values.guid}
                    disabled
                    slotProps={{
                      input: {
                        endAdornment: (
                          <InputAdornment position="end">
                            <IconButton size="small" onClick={() => onCopyToClipboard(values.guid || "")}>
                              <CopyOutlined fontSize="small" />
                            </IconButton>
                          </InputAdornment>
                        ),
                      },
                    }}
                  />
                  <NoMarginHelperText>{t("oauth.clientIdDes")}</NoMarginHelperText>
                </FormControl>
              </SettingForm>
            )}

            <SettingForm title={t("oauth.clientSecret")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required={isNew}
                  type={showSecret ? "text" : "password"}
                  value={values.secret || ""}
                  onChange={onSecretChange}
                  disabled={values.is_system}
                  placeholder={!isNew && !values.is_system ? t("oauth.secretRedactedPlaceholder") : undefined}
                  slotProps={{
                    input: {
                      endAdornment: (
                        <InputAdornment position="end">
                          {values.secret && !values.is_system && (
                            <IconButton size="small" onClick={() => onCopyToClipboard(values.secret || "")}>
                              <CopyOutlined fontSize="small" />
                            </IconButton>
                          )}
                        </InputAdornment>
                      ),
                    },
                  }}
                />
                <NoMarginHelperText>
                  {!isNew && !values.is_system ? t("oauth.clientSecretDesExisting") : t("oauth.clientSecretDes")}
                </NoMarginHelperText>
              </FormControl>
              {!values.is_system && (
                <Box sx={{ mt: 1 }}>
                  <SecondaryButton
                    variant="contained"
                    size="small"
                    startIcon={<ArrowSync />}
                    onClick={onGenerateSecret}
                  >
                    {t("oauth.generateSecret")}
                  </SecondaryButton>
                </Box>
              )}
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>

        {/* OAuth Configuration Section */}
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("oauth.oauthConfiguration")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("oauth.redirectUris")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  multiline
                  rows={4}
                  value={redirectUrisValue}
                  onChange={onRedirectUrisChange}
                  placeholder="https://example.com/callback&#10;https://localhost:8080/callback"
                  disabled={values.is_system}
                />
                <NoMarginHelperText>{t("oauth.redirectUrisDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("oauth.refreshTokenTTL")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  type="number"
                  value={values.props?.refresh_token_ttl ?? 7776000}
                  onChange={onRefreshTokenTTLChange}
                  slotProps={{
                    htmlInput: {
                      min: 0,
                    },
                    input: {
                      endAdornment: <InputAdornment position="end">{t("oauth.seconds")}</InputAdornment>,
                    },
                  }}
                />
                <NoMarginHelperText>{t("oauth.refreshTokenTTLDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>

        {/* Scopes Section */}
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("oauth.permissions")}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {t("oauth.permissionsDes")}
          </Typography>
          <SettingSectionContent>
            <BorderedCard sx={{ p: 0, overflow: "hidden" }}>
              <List dense sx={{ py: 0 }}>
                {availableScopes.map(({ scope, required, hasWrite }, index) => {
                  const scopeInfo = scopeDefinitions[scope];
                  const Icon = scopeInfo?.icon || Description;
                  const isChecked = hasScope(scope) || required;
                  const accessLevel = getAccessLevel(scope);

                  return (
                    <Box key={scope}>
                      {index > 0 && <Divider />}
                      <ListItem disablePadding>
                        <ListItemButton
                          onClick={() => !required && onScopeToggle(scope)}
                          disabled={required}
                          sx={{
                            py: 1.5,
                            borderRadius: 0,
                            flexDirection: { xs: "column", sm: "row" },
                            alignItems: { xs: "stretch", sm: "center" },
                          }}
                        >
                          <Box
                            sx={{
                              display: "flex",
                              alignItems: "center",
                              flex: 1,
                              minWidth: 0,
                            }}
                          >
                            <ListItemIcon sx={{ minWidth: 40 }}>
                              <Checkbox
                                edge="start"
                                checked={isChecked ?? false}
                                disabled={required}
                                tabIndex={-1}
                                disableRipple
                              />
                            </ListItemIcon>
                            <ListItemIcon sx={{ minWidth: 36 }}>
                              <Icon fontSize="small" color="action" />
                            </ListItemIcon>
                            <ListItemText
                              primary={
                                <Box sx={{ display: "flex", alignItems: "center", gap: 1, flexWrap: "wrap" }}>
                                  <span>
                                    {scopeInfo
                                      ? t("application:" + scopeInfo.name, { defaultValue: scope })
                                      : scope + " (OIDC)"}
                                  </span>
                                  <Chip
                                    label={hasWrite && hasWriteScope(scope) ? scope.replace(".Read", ".Write") : scope}
                                    size="small"
                                    variant="outlined"
                                    sx={{ height: 18, fontSize: "0.7rem" }}
                                  />
                                  {required && (
                                    <Chip
                                      label={t("oauth.required")}
                                      size="small"
                                      color="primary"
                                      sx={{ height: 18, fontSize: "0.7rem" }}
                                    />
                                  )}
                                </Box>
                              }
                              secondary={scopeInfo?.description ? t("application:" + scopeInfo.description) : undefined}
                              slotProps={{
                                primary: { variant: "body2" },
                                secondary: { variant: "caption" },
                              }}
                            />
                          </Box>
                          <Grow in={hasWrite && isChecked} unmountOnExit>
                            <Box
                              sx={{
                                display: "flex",
                                justifyContent: { xs: "flex-end", sm: "flex-end" },
                                mt: { xs: 1, sm: 0 },
                                ml: { xs: 0, sm: 2 },
                                pl: { xs: "76px", sm: 0 },
                              }}
                              onClick={(e) => e.stopPropagation()}
                            >
                              <ToggleButtonGroup
                                size="small"
                                value={accessLevel}
                                exclusive
                                onChange={(_, newValue) => onAccessLevelChange(scope, newValue)}
                              >
                                <ToggleButton
                                  value="read"
                                  sx={{
                                    px: 1.5,
                                    py: 0.25,
                                    fontSize: "0.75rem",
                                    textTransform: "none",
                                    borderColor: theme.palette.divider,
                                    "&.Mui-selected": {
                                      backgroundColor: alpha(theme.palette.info.main, 0.1),
                                      color: theme.palette.info.main,
                                      borderColor: theme.palette.info.main,
                                      "&:hover": {
                                        backgroundColor: alpha(theme.palette.info.main, 0.2),
                                      },
                                    },
                                  }}
                                >
                                  {t("application:oauth.scope.readonly")}
                                </ToggleButton>
                                <ToggleButton
                                  value="readwrite"
                                  sx={{
                                    px: 1.5,
                                    py: 0.25,
                                    fontSize: "0.75rem",
                                    textTransform: "none",
                                    borderColor: theme.palette.divider,
                                    "&.Mui-selected": {
                                      backgroundColor: alpha(theme.palette.warning.main, 0.1),
                                      color: theme.palette.warning.main,
                                      borderColor: theme.palette.warning.main,
                                      "&:hover": {
                                        backgroundColor: alpha(theme.palette.warning.main, 0.2),
                                      },
                                    },
                                  }}
                                >
                                  {t("application:oauth.scope.readwrite")}
                                </ToggleButton>
                              </ToggleButtonGroup>
                            </Box>
                          </Grow>
                        </ListItemButton>
                      </ListItem>
                    </Box>
                  );
                })}
              </List>
            </BorderedCard>
          </SettingSectionContent>
        </SettingSection>

        {/* Client Credentials Summary (for existing clients) */}
        {!isNew && values.guid && (
          <SettingSection>
            <Typography variant="h6" gutterBottom>
              {t("oauth.credentials")}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {t("oauth.credentialsDes")}
            </Typography>
            <SettingSectionContent>
              <SettingForm lgWidth={5}>
                <Stack spacing={2}>
                  {[
                    { label: t("oauth.clientId"), value: values.guid || "" },
                    { label: t("oauth.authorizeEndpoint"), value: `${window.location.origin}/session/authorize` },
                    { label: t("oauth.tokenEndpoint"), value: `${window.location.origin}/api/v4/session/oauth/token` },
                    {
                      label: t("oauth.refreshEndpoint"),
                      value: `${window.location.origin}/api/v4/session/token/refresh`,
                    },
                    {
                      label: t("oauth.userinfoEndpoint"),
                      value: `${window.location.origin}/api/v4/session/oauth/userinfo`,
                    },
                    { label: t("oauth.scopes"), value: (values.scopes || []).join(" "), multiline: true },
                  ].map(({ label, value, multiline }) => (
                    <DenseFilledTextField
                      key={label}
                      label={label}
                      value={value}
                      fullWidth
                      multiline={multiline}
                      rows={multiline ? 4 : undefined}
                      slotProps={{
                        input: {
                          readOnly: true,
                          endAdornment: (
                            <InputAdornment position="end">
                              <IconButton size="small" onClick={() => onCopyToClipboard(value)}>
                                <CopyOutlined fontSize="small" />
                              </IconButton>
                            </InputAdornment>
                          ),
                        },
                      }}
                    />
                  ))}
                </Stack>
              </SettingForm>
            </SettingSectionContent>
          </SettingSection>
        )}
      </Stack>
    </Box>
  );
};

export default OAuthClientForm;
