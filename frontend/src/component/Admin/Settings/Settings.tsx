import { Box, Container, FormHelperText, InputAdornment, styled } from "@mui/material";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { QueueType } from "../../../api/dashboard.ts";
import ResponsiveTabs, { Tab } from "../../Common/ResponsiveTabs.tsx";
import Bot from "../../Icons/Bot.tsx";
import Color from "../../Icons/Color.tsx";
import CubeSync from "../../Icons/CubeSync.tsx";
import Currency from "../../Icons/Currency.tsx";
import FilmstripImage from "../../Icons/FilmstripImage.tsx";
import Globe from "../../Icons/Globe.tsx";
import MailOutlined from "../../Icons/MailOutlined.tsx";
import PersonPasskey from "../../Icons/PersonPasskey.tsx";
import SendLogging from "../../Icons/SendLogging.tsx";
import Server from "../../Icons/Server.tsx";
import PageContainer from "../../Pages/PageContainer.tsx";
import PageHeader, { PageTabQuery } from "../../Pages/PageHeader.tsx";
import Appearance from "./Appearance/Appearance.tsx";
import Captcha from "./Captcha/Captcha.tsx";
import Email from "./Email/Email.tsx";
import Events from "./Event/Events.tsx";
import Media from "./Media/Media.tsx";
import Queue from "./Queue/Queue.tsx";
import ServerSetting from "./Server/ServerSetting.tsx";
import SettingsWrapper from "./SettingWrapper.tsx";
import SiteInformation from "./SiteInformation/SiteInformation.tsx";
import UserSession from "./UserSession/UserSession.tsx";
import VAS from "./VAS/VAS.tsx";

export const StyledInputAdornment = styled(InputAdornment)(({ theme }) => ({
  fontSize: theme.typography.body2.fontSize,
}));

export const SettingSection = styled(Box)(({ theme }) => ({
  display: "flex",
  flexDirection: "column",
  gap: theme.spacing(1),
  [theme.breakpoints.up("md")]: {
    padding: theme.spacing(0, 4),
  },
}));
export const SettingSectionContent = styled(Box)(({ theme }) => ({
  [theme.breakpoints.up("md")]: {
    padding: theme.spacing(0, 4),
  },
  display: "flex",
  flexDirection: "column",
  gap: theme.spacing(3),
}));
export const NoMarginHelperText = styled(FormHelperText)(() => ({
  marginLeft: 0,
  marginRight: 0,
}));

const allQueueSettings = Object.values(QueueType)
  .map((queue) => [
    `queue_${queue}_worker_num`,
    `queue_${queue}_max_execution`,
    `queue_${queue}_backoff_factor`,
    `queue_${queue}_backoff_max_duration`,
    `queue_${queue}_max_retry`,
    `queue_${queue}_retry_delay`,
  ])
  .flat();

export enum SettingsPageTab {
  SiteInformation = "siteInformation",
  UserSession = "userSession",
  Captcha = "captcha",
  FileSystem = "fileSystem",
  MediaProcessing = "mediaProcessing",
  VAS = "vas",
  Email = "email",
  Queue = "queue",
  Appearance = "appearance",
  Events = "events",
  Server = "server",
}

const Settings = () => {
  const { t } = useTranslation("dashboard");
  const [tab, setTab] = useQueryState(PageTabQuery);

  const tabs: Tab<SettingsPageTab>[] = useMemo(() => {
    const res = [];
    res.push(
      ...[
        {
          label: t("nav.basicSetting"),
          value: SettingsPageTab.SiteInformation,
          icon: <Globe />,
        },
        {
          label: t("nav.userSession"),
          value: SettingsPageTab.UserSession,
          icon: <PersonPasskey />,
        },
        {
          label: t("nav.captcha"),
          value: SettingsPageTab.Captcha,
          icon: <Bot />,
        },
        {
          label: t("nav.mediaProcessing"),
          value: SettingsPageTab.MediaProcessing,
          icon: <FilmstripImage />,
        },
        {
          label: t("vas.vas"),
          value: SettingsPageTab.VAS,
          icon: <Currency />,
        },
        {
          label: t("nav.email"),
          value: SettingsPageTab.Email,
          icon: <MailOutlined />,
        },
        {
          label: t("nav.queue"),
          value: SettingsPageTab.Queue,
          icon: <CubeSync />,
        },
        {
          label: t("nav.appearance"),
          value: SettingsPageTab.Appearance,
          icon: <Color />,
        },
        {
          label: t("nav.events"),
          value: SettingsPageTab.Events,
          icon: <SendLogging />,
        },
        {
          label: t("nav.server"),
          value: SettingsPageTab.Server,
          icon: <Server />,
        },
      ],
    );
    return res;
  }, [t]);

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.settings")} />
        <ResponsiveTabs
          value={tab ?? SettingsPageTab.SiteInformation}
          onChange={(_e, newValue) => setTab(newValue)}
          tabs={tabs}
        />
        <SwitchTransition>
          <CSSTransition
            addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
            classNames="fade"
            key={`${tab}`}
          >
            <Box>
              {(!tab || tab === SettingsPageTab.SiteInformation) && (
                <SettingsWrapper
                  settings={[
                    "siteName",
                    "siteDes",
                    "siteURL",
                    "siteScript",
                    "pwa_small_icon",
                    "pwa_medium_icon",
                    "pwa_large_icon",
                    "site_logo",
                    "site_logo_light",
                    "tos_url",
                    "privacy_policy_url",
                    "show_app_promotion",
                    "show_desktop_app_promotion",
                  ]}
                >
                  <SiteInformation />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.UserSession && (
                <SettingsWrapper
                  settings={[
                    "register_enabled",
                    "email_active",
                    "default_group",
                    "authn_enabled",
                    "expose_user_email",
                    "avatar_path",
                    "avatar_size",
                    "avatar_size_l",
                    "gravatar_server",
                  ]}
                >
                  <UserSession />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Captcha && (
                <SettingsWrapper
                  settings={[
                    "login_captcha",
                    "reg_captcha",
                    "forget_captcha",
                    "captcha_type",
                    "captcha_mode",
                    "captcha_ComplexOfNoiseText",
                    "captcha_ComplexOfNoiseDot",
                    "captcha_IsShowHollowLine",
                    "captcha_IsShowNoiseDot",
                    "captcha_IsShowNoiseText",
                    "captcha_IsShowSlimeLine",
                    "captcha_IsShowSineLine",
                    "captcha_CaptchaLen",
                    "captcha_ReCaptchaKey",
                    "captcha_ReCaptchaSecret",
                    "captcha_turnstile_site_key",
                    "captcha_turnstile_site_secret",
                    "captcha_cap_instance_url",
                    "captcha_cap_site_key",
                    "captcha_cap_secret_key",
                    "captcha_cap_asset_server",
                  ]}
                >
                  <Captcha />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.MediaProcessing && (
                <SettingsWrapper
                  settings={[
                    "thumb_width",
                    "thumb_height",
                    "thumb_entity_suffix",
                    "thumb_encode_method",
                    "thumb_gc_after_gen",
                    "thumb_encode_quality",
                    "thumb_builtin_enabled",
                    "thumb_builtin_max_size",
                    "thumb_vips_max_size",
                    "thumb_vips_enabled",
                    "thumb_vips_exts",
                    "thumb_ffmpeg_enabled",
                    "thumb_vips_path",
                    "thumb_ffmpeg_path",
                    "thumb_ffmpeg_max_size",
                    "thumb_ffmpeg_exts",
                    "thumb_ffmpeg_seek",
                    "thumb_ffmpeg_extra_args",
                    "thumb_libreoffice_path",
                    "thumb_libreoffice_max_size",
                    "thumb_libreoffice_enabled",
                    "thumb_libreoffice_exts",
                    "thumb_music_cover_enabled",
                    "thumb_music_cover_exts",
                    "thumb_music_cover_max_size",
                    "thumb_libraw_path",
                    "thumb_libraw_max_size",
                    "thumb_libraw_enabled",
                    "thumb_libraw_exts",
                    "media_meta_exif",
                    "media_meta_exif_size_local",
                    "media_meta_exif_size_remote",
                    "media_meta_exif_brute_force",
                    "media_meta_music",
                    "media_meta_music_size_local",
                    "media_exif_music_size_remote",
                    "media_meta_ffprobe",
                    "media_meta_ffprobe_path",
                    "media_meta_ffprobe_size_local",
                    "media_meta_ffprobe_size_remote",
                    "media_meta_geocoding",
                    "media_meta_geocoding_mapbox_ak",
                  ]}
                >
                  <Media />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.VAS && (
                <SettingsWrapper settings={[]}>
                  <VAS />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Email && (
                <SettingsWrapper
                  settings={[
                    "mail_keepalive",
                    "fromAdress",
                    "smtpHost",
                    "smtpPort",
                    "replyTo",
                    "smtpUser",
                    "smtpPass",
                    "smtpEncryption",
                    "fromName",
                    "mail_activation_template",
                    "mail_reset_template",
                  ]}
                >
                  <Email />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Queue && (
                <SettingsWrapper settings={allQueueSettings}>
                  <Queue />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Appearance && (
                <SettingsWrapper
                  settings={[
                    "theme_options",
                    "defaultTheme",
                    "custom_nav_items",
                    "headless_footer_html",
                    "headless_bottom_html",
                    "sidebar_bottom_html",
                  ]}
                >
                  <Appearance />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Events && (
                <SettingsWrapper settings={[]}>
                  <Events />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Server && (
                <SettingsWrapper
                  settings={[
                    "temp_path",
                    "siteID",
                    "cron_garbage_collect",
                    "hash_id_salt",
                    "access_token_ttl",
                    "refresh_token_ttl",
                  ]}
                >
                  <ServerSetting />
                </SettingsWrapper>
              )}
            </Box>
          </CSSTransition>
        </SwitchTransition>
      </Container>
    </PageContainer>
  );
};

export default Settings;
