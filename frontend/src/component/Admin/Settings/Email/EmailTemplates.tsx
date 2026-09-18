import { ExpandMoreRounded } from "@mui/icons-material";
import { Box, Typography } from "@mui/material";
import AccordionDetails from "@mui/material/AccordionDetails";
import React, { useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import SettingForm, { ProChip } from "../../../Pages/Setting/SettingForm.tsx";
import { MagicVar } from "../../Common/MagicVarDialog.tsx";
import ProDialog from "../../Common/ProDialog.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import { SettingSection, SettingSectionContent } from "../Settings.tsx";
import { AccordionSummary, StyledAccordion } from "../UserSession/SSOSettings.tsx";
import EmailTemplateEditor from "./EmailTemplateEditor.tsx";
interface EmailTemplate {
  key: string;
  title: string;
  description: string;
  magicVars: MagicVar[];
  pro: boolean;
}

const commonMagicVars: MagicVar[] = [
  {
    value: "settings.mainTitle",
    name: "{{ .CommonContext.SiteBasic.Name }}",
    example: "Cloudreve",
  },
  {
    value: "settings.siteDescription",
    name: "{{ .CommonContext.SiteBasic.Description }}",
    example: "Another Cloudreve instance",
  },
  {
    value: "settings.siteID",
    name: "{{ .CommonContext.SiteBasic.ID }}",
    example: "123e4567-e89b-12d3-a456-426614174000",
  },
  {
    value: "settings.logo",
    name: "{{ .CommonContext.Logo.Normal }}",
    example: "https://cloudreve.org/logo.svg",
  },
  {
    value: "settings.logo",
    name: "{{ .CommonContext.Logo.Light }}",
    example: "https://cloudreve.org/logo_light.svg",
  },
  {
    value: "settings.siteURL",
    name: "{{ .CommonContext.SiteUrl }}",
    example: "https://cloudreve.org",
  },
];

const userMagicVars: MagicVar[] = [
  {
    value: "policy.magicVar.uid",
    name: "{{ .User.ID }}",
    example: "2534",
  },
  {
    value: "application:login.email",
    name: "{{ .User.Email }}",
    example: "example@cloudreve.org",
  },
  {
    value: "application:setting.nickname",
    name: "{{ .User.Nick }}",
    example: "Aaron Liu",
  },
  {
    value: "user.usedStorage",
    name: "{{ .User.Storage }}",
    example: "123221000",
  },
];

const EmailTemplates: React.FC = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);
  const [proOpen, setProOpen] = useState(false);

  // Template setting keys
  const templateSettings = [
    {
      key: "mail_receipt_template",
      title: "receiptEmailTemplate",
      description: "receiptEmailTemplateDes",
      pro: true,
    },
    {
      key: "mail_activation_template",
      title: "activationEmailTemplate",
      description: "activationEmailTemplateDes",
      magicVars: [
        ...commonMagicVars,
        ...userMagicVars,
        {
          value: "settings.activateUrl",
          name: "{{ .Url }}",
          example: "https://cloudreve.org/activate",
        },
      ],
    },
    {
      key: "mail_exceed_quota_template",
      title: "quotaExceededEmailTemplate",
      description: "quotaExceededEmailTemplateDes",
      pro: true,
    },
    {
      key: "mail_reset_template",
      title: "resetPasswordEmailTemplate",
      description: "resetPasswordEmailTemplateDes",
      magicVars: [
        ...commonMagicVars,
        ...userMagicVars,
        {
          value: "settings.resetUrl",
          name: "{{ .Url }}",
          example: "https://cloudreve.org/reset",
        },
      ],
    },
  ];

  const handleProClick = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    setProOpen(true);
  };

  return (
    <SettingSection>
      <ProDialog open={proOpen} onClose={() => setProOpen(false)} />
      <Typography variant="h6" gutterBottom>
        {t("settings.emailTemplates")}
      </Typography>
      <SettingSectionContent>
        <Box>
          {templateSettings.map((template) => (
            <StyledAccordion
              disableGutters
              onClick={template.pro ? handleProClick : undefined}
              key={template.key}
              expanded={template.pro ? false : undefined}
              TransitionProps={{ unmountOnExit: true }}
            >
              <AccordionSummary expandIcon={<ExpandMoreRounded />}>
                <Typography>
                  {t("settings." + template.title)}
                  {template.pro && <ProChip label="Pro" color="primary" size="small" />}
                </Typography>
              </AccordionSummary>
              <AccordionDetails sx={{ display: "block" }}>
                <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                  {t("settings." + template.description)}{" "}
                </Typography>
                <SettingForm noContainer lgWidth={12}>
                  <Box sx={{ width: "100%" }}>
                    <EmailTemplateEditor
                      magicVars={template.magicVars || []}
                      value={values[template.key] || "[]"}
                      onChange={(value) => setSettings({ [template.key]: value })}
                      templateType={template.key}
                    />
                  </Box>
                </SettingForm>
              </AccordionDetails>
            </StyledAccordion>
          ))}
        </Box>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default EmailTemplates;
