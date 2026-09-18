import { AccordionDetailsProps, Box, styled } from "@mui/material";
import MuiAccordion, { AccordionProps } from "@mui/material/Accordion";
import MuiAccordionSummary, { AccordionSummaryProps } from "@mui/material/AccordionSummary";
import MuiAccordionDetails from "@mui/material/AccordionDetails";
import { useState } from "react";
import { CaretDownIcon } from "../FileManager/TreeView/TreeFile.tsx";
import { DefaultButton } from "../Common/StyledComponents.tsx";

const Accordion = styled((props: AccordionProps) => <MuiAccordion disableGutters elevation={0} square {...props} />)(
  ({ theme, expanded }) => ({
    borderRadius: theme.shape.borderRadius,
    backgroundColor: expanded
      ? theme.palette.mode == "light"
        ? "rgba(0, 0, 0, 0.06)"
        : "rgba(255, 255, 255, 0.09)"
      : "initial",
    "&:not(:last-child)": {
      borderBottom: 0,
    },
    "&::before": {
      display: "none",
    },
  }),
);

const AccordionSummary = styled((props: AccordionSummaryProps) => <MuiAccordionSummary {...props} />)(() => ({
  flexDirection: "row-reverse",
  minHeight: 0,
  padding: 0,
  "& .MuiAccordionSummary-content": {
    margin: 0,
  },
}));

const AccordionDetails = styled(MuiAccordionDetails)(({ theme }) => ({
  padding: theme.spacing(2),
}));

const SummaryButton = styled(DefaultButton)<{ expanded: boolean }>(({ theme, expanded }) => ({
  justifyContent: "flex-start",
  backgroundColor: expanded
    ? "initial"
    : theme.palette.mode == "light"
      ? "rgba(0, 0, 0, 0.06)"
      : "rgba(255, 255, 255, 0.09)",
  "&:hover": {
    backgroundColor: theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.09)" : "rgba(255, 255, 255, 0.13)",
  },
}));

export interface DialogAccordionProps {
  children?: React.ReactNode;
  defaultExpanded?: boolean;
  title: string;
  accordionDetailProps?: AccordionDetailsProps;
}

const DialogAccordion = (props: DialogAccordionProps) => {
  const [expanded, setExpanded] = useState(!!props.defaultExpanded);
  const handleChange = (_event: React.SyntheticEvent, newExpanded: boolean) => {
    setExpanded(newExpanded);
  };
  return (
    <Box>
      <Accordion expanded={expanded} onChange={handleChange}>
        <AccordionSummary aria-controls="panel1d-content" id="panel1d-header">
          <SummaryButton expanded={expanded} fullWidth startIcon={<CaretDownIcon expanded={expanded} />}>
            {props.title}
          </SummaryButton>
        </AccordionSummary>
        <AccordionDetails {...props.accordionDetailProps}>{props.children}</AccordionDetails>
      </Accordion>
    </Box>
  );
};

export default DialogAccordion;
