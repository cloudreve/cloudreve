import { Box, Button, Popover, PopoverProps, Stack } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { DenseFilledTextField } from "../../Common/StyledComponents";
import SettingForm from "../../Pages/Setting/SettingForm";

export interface ShareFilterPopoverProps extends PopoverProps {
  user: string;
  setUser: (user: string) => void;
  file: string;
  setFile: (file: string) => void;
  clearFilters: () => void;
}

const ShareFilterPopover = ({
  user,
  setUser,
  file,
  setFile,
  clearFilters,
  onClose,
  open,
  ...rest
}: ShareFilterPopoverProps) => {
  const { t } = useTranslation("dashboard");

  // Create local state to track changes before applying
  const [localUser, setLocalUser] = useState(user);
  const [localFile, setLocalFile] = useState(file);

  // Initialize local state when popup opens
  useEffect(() => {
    if (open) {
      setLocalUser(user);
      setLocalFile(file);
    }
  }, [open]);

  // Apply filters and close popover
  const handleApplyFilters = () => {
    setUser(localUser);
    setFile(localFile);
    onClose?.({}, "backdropClick");
  };

  // Reset filters and close popover
  const handleResetFilters = () => {
    setLocalUser("");
    setLocalFile("");
    clearFilters();
    onClose?.({}, "backdropClick");
  };

  return (
    <Popover
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "left",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "left",
      }}
      slotProps={{
        paper: {
          sx: {
            p: 2,
            width: 300,
            maxWidth: "100%",
          },
        },
      }}
      onClose={onClose}
      open={open}
      {...rest}
    >
      <Stack spacing={2}>
        <SettingForm title={t("event.userID")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localUser}
            onChange={(e) => setLocalUser(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <SettingForm title={t("event.fileID")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localFile}
            onChange={(e) => setLocalFile(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <Box display="flex" justifyContent="space-between">
          <Button variant="outlined" size="small" onClick={handleResetFilters}>
            {t("user.reset")}
          </Button>
          <Button variant="contained" size="small" onClick={handleApplyFilters}>
            {t("user.apply")}
          </Button>
        </Box>
      </Stack>
    </Popover>
  );
};

export default ShareFilterPopover;
