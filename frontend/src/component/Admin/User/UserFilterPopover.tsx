import { Box, Button, FormControl, ListItemText, Popover, PopoverProps, Stack } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { UserStatus } from "../../../api/dashboard";
import { DenseFilledTextField, DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import SettingForm from "../../Pages/Setting/SettingForm";
import GroupSelectionInput from "../Common/GroupSelectionInput";

export interface UserFilterPopoverProps extends PopoverProps {
  email: string;
  setEmail: (email: string) => void;
  nick: string;
  setNick: (nick: string) => void;
  group: string;
  setGroup: (group: string) => void;
  status: string;
  setStatus: (status: string) => void;
  clearFilters: () => void;
}

const UserFilterPopover = ({
  email,
  setEmail,
  nick,
  setNick,
  group,
  setGroup,
  status,
  setStatus,
  clearFilters,
  onClose,
  open,
  ...rest
}: UserFilterPopoverProps) => {
  const { t } = useTranslation("dashboard");

  // Create local state to track changes before applying
  const [localEmail, setLocalEmail] = useState(email);
  const [localNick, setLocalNick] = useState(nick);
  const [localGroup, setLocalGroup] = useState(group);
  const [localStatus, setLocalStatus] = useState(status);

  // Initialize local state when popup opens
  useEffect(() => {
    if (open) {
      setLocalEmail(email);
      setLocalNick(nick);
      setLocalGroup(group);
      setLocalStatus(status);
    }
  }, [open]);

  // Apply filters and close popover
  const handleApplyFilters = () => {
    setEmail(localEmail);
    setNick(localNick);
    setGroup(localGroup == " " ? "" : localGroup);
    setStatus(localStatus == " " ? "" : localStatus);
    onClose?.({}, "backdropClick");
  };

  // Reset filters and close popover
  const handleResetFilters = () => {
    setLocalEmail("");
    setLocalNick("");
    setLocalGroup("");
    setLocalStatus("");
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
        <SettingForm title={t("user.email")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localEmail}
            onChange={(e) => setLocalEmail(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <SettingForm title={t("user.nick")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localNick}
            onChange={(e) => setLocalNick(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <SettingForm title={t("user.group")} noContainer lgWidth={12}>
          <GroupSelectionInput
            value={localGroup == "" ? " " : localGroup}
            onChange={setLocalGroup}
            emptyValue={" "}
            emptyText={t("user.all")}
            fullWidth
          />
        </SettingForm>

        <SettingForm title={t("user.status")} noContainer lgWidth={12}>
          <FormControl fullWidth>
            <DenseSelect
              value={localStatus == "" ? " " : localStatus}
              onChange={(e) => setLocalStatus(e.target.value as string)}
            >
              <SquareMenuItem value=" ">
                <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                  <em>{t("user.all")}</em>
                </ListItemText>
              </SquareMenuItem>
              {Object.values(UserStatus).map((value) => (
                <SquareMenuItem value={value} key={value}>
                  <ListItemText slotProps={{ primary: { variant: "body2" } }}>{t(`user.status_${value}`)}</ListItemText>
                </SquareMenuItem>
              ))}
            </DenseSelect>
          </FormControl>
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

export default UserFilterPopover;
