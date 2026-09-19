import { Button, Dialog, DialogActions, DialogContent, DialogTitle, FormControl, ListItemText, Stack } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { batchUpdateUser } from "../../../api/api";
import { UserStatus } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import SettingForm from "../../Pages/Setting/SettingForm";
import GroupSelectionInput from "../Common/GroupSelectionInput";

export interface BatchUserDialogProps {
  open: boolean;
  onClose: () => void;
  ids: number[];
  onUpdated?: () => void;
}

const NoChange = " ";

const BatchUserDialog = ({ open, onClose, ids, onUpdated }: BatchUserDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [status, setStatus] = useState(NoChange);
  const [group, setGroup] = useState(NoChange);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setStatus(NoChange);
      setGroup(NoChange);
    }
  }, [open]);

  const onSubmit = () => {
    setLoading(true);
    dispatch(
      batchUpdateUser({
        ids,
        status: status === NoChange ? undefined : (status as "active" | "inactive" | "manual_banned"),
        group_id: group === NoChange ? undefined : parseInt(group),
      }),
    )
      .then(() => {
        onUpdated?.();
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t("user.batchEditXUsers", { num: ids.length })}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <SettingForm title={t("user.status")} noContainer lgWidth={12}>
            <FormControl fullWidth>
              <DenseSelect value={status} onChange={(e) => setStatus(e.target.value as string)}>
                <SquareMenuItem value={NoChange}>
                  <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                    <em>{t("user.noChange")}</em>
                  </ListItemText>
                </SquareMenuItem>
                {Object.values(UserStatus)
                  .filter((value) => value !== UserStatus.sys_banned)
                  .map((value) => (
                  <SquareMenuItem value={value} key={value}>
                    <ListItemText slotProps={{ primary: { variant: "body2" } }}>{t(`user.status_${value}`)}</ListItemText>
                  </SquareMenuItem>
                ))}
              </DenseSelect>
            </FormControl>
          </SettingForm>
          <SettingForm title={t("user.group")} noContainer lgWidth={12}>
            <GroupSelectionInput
              value={group}
              onChange={setGroup}
              emptyValue={NoChange}
              emptyText={t("user.noChange")}
              fullWidth
            />
          </SettingForm>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button
          variant="contained"
          onClick={onSubmit}
          disabled={loading || ids.length === 0 || (status === NoChange && group === NoChange)}
        >
          {t("user.apply")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BatchUserDialog;
