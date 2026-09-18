import { Delete } from "@mui/icons-material";
import {
  Box,
  Button,
  DialogContent,
  FormControl,
  Link,
  ListItemText,
  Tab,
  Tabs,
  Typography,
  useTheme,
} from "@mui/material";
import React, { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { languages } from "../../../../i18n.ts";
import CircularProgress from "../../../Common/CircularProgress.tsx";
import { DenseFilledTextField, DenseSelect, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import MagicVarDialog from "../../Common/MagicVarDialog.tsx";
import { NoMarginHelperText } from "../Settings.tsx";

const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor.tsx"));

interface TemplateItem {
  language: string;
  title: string;
  body: string;
}

interface EmailTemplateEditorProps {
  value: string;
  onChange: (value: string) => void;
  templateType: string;
  magicVars: MagicVar[];
}

const EmailTemplateEditor: React.FC<EmailTemplateEditorProps> = ({ value, onChange, templateType, magicVars }) => {
  const theme = useTheme();
  const { t } = useTranslation("dashboard");
  const [templates, setTemplates] = useState<TemplateItem[]>([]);
  const [currentTab, setCurrentTab] = useState(0);
  const [addLanguageOpen, setAddLanguageOpen] = useState(false);
  const [newLanguageCode, setNewLanguageCode] = useState("");
  const isUpdatingFromProp = useRef(false);
  const [magicVarOpen, setMagicVarOpen] = useState(false);

  // Parse templates when component mounts or value changes
  useEffect(() => {
    try {
      isUpdatingFromProp.current = true;
      const parsedTemplates = value ? JSON.parse(value) : [];
      setTemplates(parsedTemplates);
      // If no templates, create a default English one
      if (parsedTemplates.length === 0) {
        setTemplates([{ language: "en-US", title: "", body: "" }]);
      }
      if (currentTab > parsedTemplates.length) {
        setCurrentTab(0);
      }
    } catch (e) {
      console.error("Failed to parse email template:", e);
      setTemplates([{ language: "en-US", title: "", body: "" }]);
    } finally {
      // Use setTimeout to ensure this runs after React finishes the update
      setTimeout(() => {
        isUpdatingFromProp.current = true; // Prevent infinite loop
      }, 0);
    }
  }, [value]);

  // Update the parent component when templates change, but only from user interaction
  useEffect(() => {
    if (templates.length > 0 && !isUpdatingFromProp.current) {
      onChange(JSON.stringify(templates));
    }
  }, [templates, onChange]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setCurrentTab(newValue);
  };

  const updateTemplate = (index: number, field: keyof TemplateItem, newValue: string) => {
    isUpdatingFromProp.current = false; // Ensure this is a user interaction
    const updatedTemplates = [...templates];
    updatedTemplates[index] = {
      ...updatedTemplates[index],
      [field]: newValue,
    };
    setTemplates(updatedTemplates);
  };

  const addNewLanguage = () => {
    if (!newLanguageCode.trim()) return;

    // Check if language already exists
    const langTemplateIndex = templates.findIndex((l) => l.language === newLanguageCode);
    if (langTemplateIndex !== -1) {
      setNewLanguageCode("");
      setAddLanguageOpen(false);

      setCurrentTab(langTemplateIndex);
      return;
    }

    // Add new language template
    isUpdatingFromProp.current = false; // Ensure this is a user interaction
    setTemplates([...templates, { language: newLanguageCode, title: "", body: "" }]);

    // Reset and close dialog
    setNewLanguageCode("");
    setAddLanguageOpen(false);

    // Switch to the new tab
    setCurrentTab(templates.length);
  };

  const removeLanguage = (index: number) => {
    isUpdatingFromProp.current = false; // Ensure this is a user interaction
    const updatedTemplates = templates.filter((_, i) => i !== index);
    setTemplates(updatedTemplates);

    if (currentTab >= updatedTemplates.length) {
      setCurrentTab(updatedTemplates.length - 1); // Move to the last tab if current is out of range
    }
  };

  const setPreferredLanguage = (index: number) => {
    isUpdatingFromProp.current = false; // Ensure this is a user interaction
    setTemplates([templates[index], ...templates.filter((_, i) => i !== index)]);
    setCurrentTab(0); // Switch to the first tab as the preferred language is now at the top
  };

  const openMagicVar = useCallback((e: React.MouseEvent<HTMLElement>) => {
    setMagicVarOpen(true);
    e.stopPropagation();
    e.preventDefault();
  }, []);

  return (
    <Box sx={{ width: "100%" }}>
      <Box sx={{ borderBottom: 1, borderColor: "divider", display: "flex" }}>
        <Tabs
          value={currentTab}
          onChange={handleTabChange}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ flexGrow: 1 }}
        >
          {templates.map((template, index) => {
            const lang = languages.find((l) => l.code === template.language);
            return <Tab key={index} label={lang ? lang.displayName : template.language} />;
          })}
        </Tabs>
        <Button
          startIcon={<Add />}
          onClick={() => setAddLanguageOpen(true)}
          sx={{ minWidth: "auto", mt: 0.5, mb: 0.5 }}
        >
          {t("settings.addLanguage")}
        </Button>
      </Box>
      {templates.map((template, index) => (
        <Box
          key={index}
          role="tabpanel"
          hidden={currentTab !== index}
          id={`template-tabpanel-${index}`}
          aria-labelledby={`template-tab-${index}`}
          sx={{ mt: 2 }}
        >
          {currentTab === index && (
            <Box>
              <FormControl fullWidth sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: index === 0 ? 0 : 1 }}>
                  {t("settings.preferredLanguage")}
                </Typography>
                {index != 0 && (
                  <Box>
                    <SecondaryButton
                      variant="contained"
                      onClick={() => (setPreferredLanguage(index))}
                    >
                      {t("settings.setAsPreferredLanguage")}
                    </SecondaryButton>
                  </Box>
                )}
                <NoMarginHelperText>
                  {t(index === 0 ? "settings.alreadyAsPreferredLanguageDes" : "settings.setAsPreferredLanguageDes")}
                </NoMarginHelperText>
              </FormControl>

              <FormControl fullWidth sx={{ mb: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  {t("settings.emailSubject")}
                </Typography>
                <DenseFilledTextField
                  fullWidth
                  value={template.title}
                  onChange={(e) => updateTemplate(index, "title", e.target.value || "")}
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey={"settings.emailSubjectDes"}
                    ns={"dashboard"}
                    components={[<Link onClick={openMagicVar} href={"#"} />]}
                  />
                </NoMarginHelperText>
              </FormControl>

              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                {t("settings.emailBody")}
              </Typography>
              <Box sx={{ height: 400 }}>
                <Suspense fallback={<CircularProgress />}>
                  <MonacoEditor
                    theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                    language="html"
                    value={template.body}
                    onChange={(value) => updateTemplate(index, "body", value || "")}
                    height="400px"
                    minHeight="400px"
                    options={{
                      wordWrap: "on",
                      minimap: { enabled: false },
                      scrollBeyondLastLine: false,
                    }}
                  />
                </Suspense>
              </Box>
              <NoMarginHelperText sx={{ mb: 2 }}>
                <Trans
                  i18nKey={"settings.emailBodyDes"}
                  ns={"dashboard"}
                  components={[<Link onClick={openMagicVar} href={"#"} />]}
                />
              </NoMarginHelperText>

              <FormControl fullWidth>
                <Typography variant="subtitle2" sx={{ mb: index === 0 ? 0 : 1 }}>
                  {t("settings.removeLanguage")}
                </Typography>
                {index != 0 && (
                  <Box>
                    <Button
                      startIcon={<Delete />}
                      variant="contained"
                      color="error"
                      onClick={() => removeLanguage(index)}
                    >
                      {t("settings.removeLanguageBtn")}
                    </Button>
                  </Box>
                )}
                {index === 0 && (
                  <NoMarginHelperText>
                    {t("settings.cannotRemovePreferredLanguageDes")}
                  </NoMarginHelperText>
                )}
              </FormControl>
            </Box>
          )}
        </Box>
      ))}
      {/* Add Language Dialog */}
      <DraggableDialog
        title={t("settings.addLanguage")}
        showActions
        showCancel
        onAccept={addNewLanguage}
        dialogProps={{
          maxWidth: "xs",
          fullWidth: true,
          open: addLanguageOpen,
          onClose: () => setAddLanguageOpen(false),
        }}
      >
        <DialogContent>
          <SettingForm title={t("application:setting.language")} lgWidth={12}>
            <FormControl fullWidth>
              <DenseSelect value={newLanguageCode} onChange={(e) => setNewLanguageCode(e.target.value as string)}>
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
              <NoMarginHelperText>{t("settings.languageCodeDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
        </DialogContent>
      </DraggableDialog>
      <MagicVarDialog open={magicVarOpen} vars={magicVars} onClose={() => setMagicVarOpen(false)} />
    </Box>
  );
};

export default EmailTemplateEditor;
