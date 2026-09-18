import { ExpandMoreRounded } from "@mui/icons-material";
import { Accordion, AccordionDetails, FormControlLabel, styled } from "@mui/material";
import MuiAccordionSummary, { AccordionSummaryProps } from "@mui/material/AccordionSummary";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { StyledCheckbox } from "../../../Common/StyledComponents.tsx";
import ProDialog from "../../Common/ProDialog.tsx";

export const AccordionSummary = styled((props: AccordionSummaryProps) => <MuiAccordionSummary {...props} />)(
  ({ theme }) => ({
    fontSize: theme.typography.body2.fontSize,
    paddingLeft: theme.spacing(4),
    "& .MuiFormControlLabel-label": {
      fontSize: theme.typography.body2.fontSize,
    },
    "& .MuiCheckbox-root": {
      marginRight: theme.spacing(2),
    },
  }),
);

export const StyledAccordion = styled(Accordion)(({ theme }) => ({
  boxShadow: "none",
  border: `1px solid ${theme.palette.divider}`,
  "&::before": {
    display: "none",
  },
}));

export interface SettingSectionProps {}

const SSOSettings = () => {
  const [open, setOpen] = useState(false);
  const { t } = useTranslation("dashboard");
  const onClick = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    setOpen(true);
  }, []);
  return (
    <>
      <ProDialog open={open} onClose={() => setOpen(false)} />
      <div onClick={onClick}>
        <StyledAccordion expanded={false} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreRounded />}>
            <FormControlLabel control={<StyledCheckbox size={"small"} checked={false} />} label={t("vas.qqConnect")} />
          </AccordionSummary>
          <AccordionDetails sx={{ display: "block" }}></AccordionDetails>
        </StyledAccordion>
        <StyledAccordion expanded={false} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreRounded />}>
            <FormControlLabel control={<StyledCheckbox size={"small"} checked={false} />} label={t("settings.logto")} />
          </AccordionSummary>
          <AccordionDetails sx={{ display: "block" }}></AccordionDetails>
        </StyledAccordion>
        <StyledAccordion expanded={false} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreRounded />}>
            <FormControlLabel control={<StyledCheckbox size={"small"} checked={false} />} label={t("settings.oidc")} />
          </AccordionSummary>
          <AccordionDetails sx={{ display: "block" }}></AccordionDetails>
        </StyledAccordion>
      </div>
    </>
  );
};

export default SSOSettings;
