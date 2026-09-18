import {
  Box,
  Checkbox,
  Divider,
  FormControl,
  FormControlLabel,
  FormGroup,
  Grid,
  Stack,
  Typography,
} from "@mui/material";
import { useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { AuditLogType } from "../../../../api/explorer";
import { ProChip } from "../../../Pages/Setting/SettingForm";
import ProDialog from "../../Common/ProDialog";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings";
import { SettingContext } from "../SettingWrapper";

// Categorize audit events
export const eventCategories = {
  system: {
    title: "settings.systemEvents",
    description: "settings.systemEventsDes",
    events: [AuditLogType.server_start],
  },
  user: {
    title: "settings.userEvents",
    description: "settings.userEventsDes",
    events: [
      AuditLogType.user_signup,
      AuditLogType.user_activated,
      AuditLogType.user_login,
      AuditLogType.user_login_failed,
      AuditLogType.user_token_refresh,
      AuditLogType.user_changed,
      AuditLogType.user_exceed_quota_notified,
      AuditLogType.change_nick,
      AuditLogType.change_avatar,
      AuditLogType.change_password,
      AuditLogType.enable_2fa,
      AuditLogType.disable_2fa,
      AuditLogType.add_passkey,
      AuditLogType.remove_passkey,
      AuditLogType.link_account,
      AuditLogType.unlink_account,
      AuditLogType.report_abuse,
      AuditLogType.oauth_grant_create,
      AuditLogType.oauth_token_exchange,
      AuditLogType.oauth_grant_revoke,
    ],
  },
  file: {
    title: "settings.fileEvents",
    description: "settings.fileEventsDes",
    events: [
      AuditLogType.file_create,
      AuditLogType.file_imported,
      AuditLogType.file_rename,
      AuditLogType.set_file_permission,
      AuditLogType.entity_uploaded,
      AuditLogType.entity_downloaded,
      AuditLogType.copy_from,
      AuditLogType.copy_to,
      AuditLogType.move_to,
      AuditLogType.delete_file,
      AuditLogType.move_to_trash,
      AuditLogType.update_metadata,
      AuditLogType.get_direct_link,
      AuditLogType.delete_direct_link,
      AuditLogType.update_view,
    ],
  },
  share: {
    title: "settings.shareEvents",
    description: "settings.shareEventsDes",
    events: [AuditLogType.share, AuditLogType.share_link_viewed, AuditLogType.edit_share, AuditLogType.delete_share],
  },
  version: {
    title: "settings.versionEvents",
    description: "settings.versionEventsDes",
    events: [AuditLogType.set_current_version, AuditLogType.delete_version],
  },
  media: {
    title: "settings.mediaEvents",
    description: "settings.mediaEventsDes",
    events: [AuditLogType.thumb_generated, AuditLogType.live_photo_uploaded],
  },
  filesystem: {
    title: "settings.filesystemEvents",
    description: "settings.filesystemEventsDes",
    events: [AuditLogType.mount, AuditLogType.relocate, AuditLogType.create_archive, AuditLogType.extract_archive],
  },
  webdav: {
    title: "settings.webdavEvents",
    description: "settings.webdavEventsDes",
    events: [
      AuditLogType.webdav_login_failed,
      AuditLogType.webdav_account_create,
      AuditLogType.webdav_account_update,
      AuditLogType.webdav_account_delete,
    ],
  },
  payment: {
    title: "settings.paymentEvents",
    description: "settings.paymentEventsDes",
    events: [
      AuditLogType.payment_created,
      AuditLogType.points_change,
      AuditLogType.payment_paid,
      AuditLogType.payment_fulfilled,
      AuditLogType.payment_fulfill_failed,
      AuditLogType.storage_added,
      AuditLogType.group_changed,
      AuditLogType.membership_unsubscribe,
      AuditLogType.redeem_gift_code,
    ],
  },
  email: {
    title: "settings.emailEvents",
    description: "settings.emailEventsDes",
    events: [AuditLogType.email_sent],
  },
};

// Get event name from AuditLogType
export const getEventName = (eventType: number): string => {
  return Object.entries(AuditLogType).find(([_, value]) => value === eventType)?.[0] || `event_${eventType}`;
};

const Events = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);
  const [proOpen, setProOpen] = useState(false);

  const handleProClick = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    setProOpen(true);
  };

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <ProDialog open={proOpen} onClose={() => setProOpen(false)} />
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.auditLog")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {t("settings.auditLogDes")}
          </Typography>

          {Object.entries(eventCategories).map(([categoryKey, category]) => (
            <SettingSection key={categoryKey} onClick={handleProClick}>
              <Box>
                <Typography variant="subtitle1">{t(category.title)}</Typography>
                <Typography variant="body2" color="text.secondary">
                  {t(category.description)}
                </Typography>
              </Box>

              <SettingSectionContent>
                <FormControl component="fieldset">
                  <FormGroup>
                    <FormControlLabel
                      slotProps={{
                        typography: {
                          variant: "body2",
                        },
                      }}
                      control={<Checkbox size={"small"} checked={false} />}
                      label={t("settings.toggleAll")}
                    />
                    <NoMarginHelperText>{t("settings.toggleAllDes")}</NoMarginHelperText>
                  </FormGroup>
                </FormControl>
                <Grid container spacing={1}>
                  {category.events.map((eventType) => (
                    <Grid item xs={12} sm={6} md={4} lg={3} key={eventType}>
                      <FormControlLabel
                        slotProps={{
                          typography: {
                            variant: "body2",
                          },
                        }}
                        control={<Checkbox size={"small"} checked={false} />}
                        label={t(`settings.event.${getEventName(eventType)}`, getEventName(eventType))}
                      />
                    </Grid>
                  ))}
                </Grid>
              </SettingSectionContent>
              <Divider sx={{ mt: 1 }} />
            </SettingSection>
          ))}
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default Events;
