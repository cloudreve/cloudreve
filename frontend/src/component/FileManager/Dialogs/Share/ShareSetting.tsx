import {
  Autocomplete,
  Checkbox,
  Collapse,
  createFilterOptions,
  FormControl,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemSecondaryAction,
  ListItemText,
  Stack,
  styled,
  TextField,
  Typography,
} from "@mui/material";
import MuiAccordion from "@mui/material/Accordion";
import MuiAccordionDetails from "@mui/material/AccordionDetails";
import MuiAccordionSummary from "@mui/material/AccordionSummary";
import { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { FileResponse, FileType } from "../../../../api/explorer.ts";
import { Code } from "../../../Common/Code.tsx";
import { FilledTextField, SmallFormControlLabel } from "../../../Common/StyledComponents.tsx";
import BookInformation from "../../../Icons/BookInformation.tsx";
import ClockArrowDownload from "../../../Icons/ClockArrowDownload.tsx";
import Eye from "../../../Icons/Eye.tsx";
import TableSettingsOutlined from "../../../Icons/TableSettings.tsx";
import Timer from "../../../Icons/Timer.tsx";

const Accordion = styled(MuiAccordion)(() => ({
  border: "0px solid rgba(0, 0, 0, .125)",
  boxShadow: "none",
  "&:not(:last-child)": {
    borderBottom: 0,
  },
  "&:before": {
    display: "none",
  },
  ".Mui-expanded": {
    margin: "0 0",
    minHeight: 0,
  },
  "&.Mui-expanded": {
    margin: "0 0",
    minHeight: 0,
  },
}));

const AccordionSummary = styled(MuiAccordionSummary)(({ theme }) => ({
  padding: 0,
  "& .MuiAccordionSummary-content": {
    margin: 0,
    display: "initial",
    "&.Mui-expanded": {
      margin: "0 0",
    },
  },
  "&.Mui-expanded": {
    borderRadius: "12px 12px 0 0",
    backgroundColor: theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.06)" : "rgba(255, 255, 255, 0.09)",
    minHeight: "0px!important",
  },
}));

const AccordionDetails = styled(MuiAccordionDetails)(({ theme }) => ({
  padding: 24,
  backgroundColor: theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.06)" : "rgba(255, 255, 255, 0.09)",
  borderRadius: "0 0 12px 12px",
  fontSize: theme.typography.body2.fontSize,
  color: theme.palette.text.secondary,
}));

const StyledListItemButton = styled(ListItemButton)(() => ({}));

export interface ShareSetting {
  is_private?: boolean;
  use_custom_password?: boolean;
  password?: string;
  share_view?: boolean;
  show_readme?: boolean;
  downloads?: boolean;
  expires?: boolean;

  downloads_val: valueOption;
  expires_val: valueOption;
}

export interface ShareSettingProps {
  setting: ShareSetting;
  file?: FileResponse;
  onSettingChange: (value: ShareSetting) => void;
  editing?: boolean;
}

interface valueOption {
  value: number;
  label: string;
  inputValue?: string;
}

export const expireOptions: valueOption[] = [
  { value: 300, label: "modals.5minutes" },
  { value: 3600, label: "modals.1hour" },
  { value: 24 * 3600, label: "modals.1day" },
  { value: 7 * 24 * 3600, label: "modals.7days" },
  { value: 30 * 24 * 3600, label: "modals.30days" },
];

export const downloadOptions: valueOption[] = [
  { value: 1, label: "1" },
  { value: 2, label: "2" },
  { value: 3, label: "3" },
  { value: 4, label: "4" },
  { value: 5, label: "5" },
  { value: 20, label: "20" },
  { value: 50, label: "50" },
  { value: 100, label: "100" },
];

const isNumeric = (num: any) =>
  (typeof num === "number" || (typeof num === "string" && num.trim() !== "")) && !isNaN(num as number);

const filter = createFilterOptions<valueOption>();

const ShareSettingContent = ({ setting, file, editing, onSettingChange }: ShareSettingProps) => {
  const { t } = useTranslation();

  const [expanded, setExpanded] = useState<string | undefined>(undefined);

  const handleExpand = (panel: string) => (_event: any, isExpanded: boolean) => {
    setExpanded(isExpanded ? panel : undefined);
  };

  const handleCheck = (prop: "is_private" | "share_view" | "show_readme" | "expires" | "downloads") => () => {
    if (!setting[prop]) {
      handleExpand(prop)(null, true);
    }

    onSettingChange({ ...setting, [prop]: !setting[prop] });
  };

  return (
    <List
      sx={{
        padding: 0,
      }}
    >
      <Accordion expanded={expanded === "is_private"} onChange={handleExpand("is_private")}>
        <AccordionSummary aria-controls="panel1a-content" id="panel1a-header">
          <StyledListItemButton>
            <ListItemIcon>
              <Eye />
            </ListItemIcon>
            <ListItemText primary={t("application:modals.privateShare")} />
            <ListItemSecondaryAction>
              <Checkbox disabled={editing} checked={!!setting.is_private} onChange={handleCheck("is_private")} />
            </ListItemSecondaryAction>
          </StyledListItemButton>
        </AccordionSummary>
        <AccordionDetails sx={{ display: "flex", flexDirection: "column", alignItems: "center" }}>
          <Typography variant="body2">{t("application:modals.privateShareDes")}</Typography>
          {setting.is_private && (
            <Stack sx={{ mt: 1, width: "100%" }}>
              {!editing && (
                <SmallFormControlLabel
                  control={
                    <Checkbox
                      size="small"
                      checked={setting.use_custom_password}
                      onChange={() => {
                        onSettingChange({ ...setting, use_custom_password: !setting.use_custom_password });
                      }}
                    />
                  }
                  label={t("application:modals.useCustomPassword")}
                />
              )}
              <Collapse in={setting.use_custom_password}>
                <FormControl variant="standard" fullWidth sx={{ mt: 1 }}>
                  <FilledTextField
                    label={t("application:modals.sharePassword")}
                    disabled={editing}
                    slotProps={{
                      htmlInput: {
                        maxLength: 32,
                      },
                    }}
                    value={setting.password ?? ""}
                    onChange={(e) => {
                      const value = e.target.value.trim();
                      if (!/^[a-zA-Z0-9]*$/.test(value) || value.length > 32) return;
                      onSettingChange({ ...setting, password: value });
                    }}
                    required
                  />
                </FormControl>
              </Collapse>
            </Stack>
          )}
        </AccordionDetails>
      </Accordion>
      {file?.type == FileType.folder && (
        <>
          <Accordion expanded={expanded === "share_view"} onChange={handleExpand("share_view")}>
            <AccordionSummary aria-controls="panel1a-content" id="panel1a-header">
              <StyledListItemButton>
                <ListItemIcon>
                  <TableSettingsOutlined />
                </ListItemIcon>
                <ListItemText primary={t("application:modals.shareView")} />
                <ListItemSecondaryAction>
                  <Checkbox checked={setting.share_view} onChange={handleCheck("share_view")} />
                </ListItemSecondaryAction>
              </StyledListItemButton>
            </AccordionSummary>
            <AccordionDetails>{t("application:modals.shareViewDes")}</AccordionDetails>
          </Accordion>
          <Accordion expanded={expanded === "show_readme"} onChange={handleExpand("show_readme")}>
            <AccordionSummary aria-controls="panel1a-content" id="panel1a-header">
              <StyledListItemButton>
                <ListItemIcon>
                  <BookInformation />
                </ListItemIcon>
                <ListItemText primary={t("application:modals.showReadme")} />
                <ListItemSecondaryAction>
                  <Checkbox checked={setting.show_readme} onChange={handleCheck("show_readme")} />
                </ListItemSecondaryAction>
              </StyledListItemButton>
            </AccordionSummary>
            <AccordionDetails>
              <Trans i18nKey="application:modals.showReadmeDes" components={[<Code />]} />
            </AccordionDetails>
          </Accordion>
        </>
      )}
      <Accordion expanded={expanded === "expires"} onChange={handleExpand("expires")}>
        <AccordionSummary aria-controls="panel1a-content" id="panel1a-header">
          <StyledListItemButton>
            <ListItemIcon>
              <Timer />
            </ListItemIcon>
            <ListItemText primary={t("modals.expireAutomatically")} />
            <ListItemSecondaryAction>
              <Checkbox checked={setting.expires} onChange={handleCheck("expires")} />
            </ListItemSecondaryAction>
          </StyledListItemButton>
        </AccordionSummary>
        <AccordionDetails sx={{ display: "flex", alignItems: "center" }}>
          <Typography>{t("application:modals.expirePrefix")}</Typography>
          <FormControl
            variant="standard"
            style={{
              marginRight: 10,
              marginLeft: 10,
            }}
          >
            <Autocomplete
              value={setting.expires_val}
              filterOptions={(options, params) => {
                const filtered = filter(options, params);

                const { inputValue } = params;
                const value = parseInt(inputValue) * 60;
                if (inputValue !== "" && isNumeric(inputValue) && parseInt(inputValue) > 0 && value != 300) {
                  filtered.push({
                    inputValue,
                    value,
                    label: inputValue + " " + t("application:modals.minutes"),
                  });
                }

                return filtered;
              }}
              onChange={(_event, newValue) => {
                let expiry = 0;
                let label = "";
                if (typeof newValue === "string") {
                  expiry = parseInt(newValue);
                  label = newValue + " " + t("application:modals.minutes");
                } else {
                  expiry = newValue?.value ?? 0;
                  label = newValue?.label ?? "";
                }

                onSettingChange({
                  ...setting,
                  expires_val: { value: expiry, label },
                });
              }}
              freeSolo
              getOptionLabel={(option: string | valueOption) => (typeof option === "string" ? option : t(option.label))}
              disableClearable
              options={expireOptions}
              renderInput={(params) => <TextField sx={{ width: 150 }} {...params} variant={"standard"} />}
            />
          </FormControl>
          <Typography>{t("application:modals.expireSuffix")}</Typography>
        </AccordionDetails>
      </Accordion>
      {file?.type == FileType.file && (
        <Accordion expanded={expanded === "downloads"} onChange={handleExpand("downloads")}>
          <AccordionSummary aria-controls="panel1a-content" id="panel1a-header">
            <StyledListItemButton>
              <ListItemIcon>
                <ClockArrowDownload />
              </ListItemIcon>
              <ListItemText primary={t("application:modals.expireAfterDownload")} />
              <ListItemSecondaryAction>
                <Checkbox checked={setting.downloads} onChange={handleCheck("downloads")} />
              </ListItemSecondaryAction>
            </StyledListItemButton>
          </AccordionSummary>
          <AccordionDetails sx={{ display: "flex", alignItems: "center" }}>
            <Typography>{t("application:modals.expirePrefix")}</Typography>
            <FormControl
              variant="standard"
              style={{
                marginRight: 10,
                marginLeft: 10,
              }}
            >
              <Autocomplete
                value={setting.downloads_val}
                filterOptions={(options, params) => {
                  const filtered = filter(options, params);

                  const { inputValue } = params;
                  const value = parseInt(inputValue);
                  if (
                    inputValue !== "" &&
                    isNumeric(inputValue) &&
                    parseInt(inputValue) > 0 &&
                    !filtered.find((v) => v.value == value)
                  ) {
                    filtered.push({
                      inputValue,
                      value,
                      label: inputValue,
                    });
                  }

                  return filtered;
                }}
                onChange={(_event, newValue) => {
                  let downloads = 0;
                  let label = "";
                  if (typeof newValue === "string") {
                    downloads = parseInt(newValue);
                    label = newValue;
                  } else {
                    downloads = newValue?.value ?? 0;
                    label = newValue?.label ?? "";
                  }

                  onSettingChange({
                    ...setting,
                    downloads_val: { value: downloads, label },
                  });
                }}
                freeSolo
                getOptionLabel={(option: string | valueOption) =>
                  typeof option === "string"
                    ? option
                    : t("application:modals.downloadLimitOptions", {
                        num: option.label,
                      })
                }
                disableClearable
                options={downloadOptions}
                renderInput={(params) => <TextField sx={{ width: 200 }} {...params} variant={"standard"} />}
              />
            </FormControl>
            <Typography>{t("application:modals.expireSuffix")}</Typography>
          </AccordionDetails>
        </Accordion>
      )}
    </List>
  );
};

export default ShareSettingContent;
