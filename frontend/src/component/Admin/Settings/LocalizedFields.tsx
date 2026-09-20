import { Accordion, AccordionDetails, AccordionSummary, Stack, Typography } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { languages } from "../../../i18n.ts";
import { DenseFilledTextField } from "../../Common/StyledComponents.tsx";
import CaretDown from "../../Icons/CaretDown.tsx";

export interface LocalizedFieldsProps {
  // JSON-encoded { "<lang>": "text" } map; "" or "{}" means no overrides.
  value?: string;
  onChange: (json: string) => void;
  multiline?: boolean;
  rows?: number;
}

// LocalizedFields edits the "<key>_i18n" sibling map of a base setting: one
// input per supported language, empty inputs are dropped from the map so the
// base value stays the fallback.
const LocalizedFields = ({ value, onChange, multiline, rows }: LocalizedFieldsProps) => {
  const { t } = useTranslation("dashboard");

  const map = useMemo(() => {
    if (!value) {
      return {} as { [k: string]: string };
    }
    try {
      const parsed = JSON.parse(value);
      return typeof parsed === "object" && parsed !== null ? (parsed as { [k: string]: string }) : {};
    } catch {
      return {};
    }
  }, [value]);

  const setLang = (code: string, text: string) => {
    const next = { ...map, [code]: text };
    if (!text) {
      delete next[code];
    }
    onChange(JSON.stringify(next));
  };

  return (
    <Accordion
      disableGutters
      elevation={0}
      sx={{
        mt: 1,
        "&:before": { display: "none" },
        border: (theme) => `1px dashed ${theme.palette.divider}`,
        borderRadius: 1,
      }}
    >
      <AccordionSummary expandIcon={<CaretDown />} sx={{ minHeight: 40 }}>
        <Typography variant="body2" color="text.secondary">
          {t("settings.translations")}
        </Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Stack spacing={1.5}>
          {languages.map((l) => (
            <DenseFilledTextField
              key={l.code}
              fullWidth
              size="small"
              label={l.displayName}
              value={map[l.code] ?? ""}
              multiline={multiline}
              rows={multiline ? rows : undefined}
              onChange={(e) => setLang(l.code, e.target.value)}
            />
          ))}
          <Typography variant="caption" color="text.secondary">
            {t("settings.translationsDes")}
          </Typography>
        </Stack>
      </AccordionDetails>
    </Accordion>
  );
};

export default LocalizedFields;
