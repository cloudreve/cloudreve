import {
  Card,
  CardActionArea,
  CardContent,
  CardMedia,
  DialogActions,
  DialogContent,
  Grid2,
  styled,
  Typography,
} from "@mui/material";
import { useTranslation } from "react-i18next";
import { PolicyType } from "../../../api/explorer";
import { SecondaryButton } from "../../Common/StyledComponents";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import Open from "../../Icons/Open";
import { PolicyPropsMap } from "./StoragePolicySetting";
import { useState } from "react";
import ProDialog from "../Common/ProDialog.tsx";
import { ProChip } from "../../Pages/Setting/SettingForm.tsx";

export interface SelectProviderProps {
  open: boolean;
  onClose: () => void;
  onSelect: (provider: PolicyType) => void;
}

const StyledCard = styled(Card)(({ theme }) => ({
  display: "flex",
  boxShadow: "none",
  border: `1px solid ${theme.palette.divider}`,
}));

const SelectProvider = ({ open, onClose, onSelect }: SelectProviderProps) => {
  const { t } = useTranslation("dashboard");
  const [proOpen, setProOpen] = useState(false);
  return (
    <DraggableDialog
      title={t("policy.selectAStorageProvider")}
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <ProDialog open={proOpen} onClose={() => setProOpen(false)} />
      <DialogContent dividers>
        <Grid2 container spacing={2} sx={{ mt: 2 }}>
          {Object.values(PolicyType).map((type) => (
            <Grid2 key={type.toString()} size={{ sm: 12, md: 6 }}>
              <StyledCard sx={{ display: "flex" }}>
                <CardActionArea
                  sx={{ display: "flex", justifyContent: "flex-start" }}
                  onClick={() => (PolicyPropsMap[type].pro ? setProOpen(true) : onSelect(type))}
                >
                  <CardMedia component="img" image={PolicyPropsMap[type].img} sx={{ width: 100, height: 60 }} />
                  <CardContent>
                    <Typography variant="subtitle1" color="textSecondary">
                      {t(PolicyPropsMap[type].name)}
                      {PolicyPropsMap[type].pro && <ProChip size="small" label="Pro" />}
                    </Typography>
                  </CardContent>
                </CardActionArea>
              </StyledCard>
            </Grid2>
          ))}
        </Grid2>
      </DialogContent>
      <DialogActions sx={{ justifyContent: "flex-start", p: 2, px: 3 }}>
        <SecondaryButton
          startIcon={<Open />}
          variant="contained"
          onClick={() => window.open("https://docs.cloudreve.org/usage/storage/", "_blank")}
        >
          {t("policy.compare")}
        </SecondaryButton>
      </DialogActions>
    </DraggableDialog>
  );
};

export default SelectProvider;
