import { DialogContent, Stack } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { upsertUser } from "../../../api/api";
import { User, UserStatus } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseFilledTextField } from "../../Common/StyledComponents";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import SettingForm from "../../Pages/Setting/SettingForm";
import GroupSelectionInput from "../Common/GroupSelectionInput";

export interface NewUserDialogProps {
  open: boolean;
  onClose: () => void;
  onCreated: (user: User) => void;
}

const defaultUser: User = {
  edges: {},
  id: 0,
  email: "",
  nick: "",
  password: "",
  status: UserStatus.active,
  group_users: 2,
};

const NewUserDialog = ({ open, onClose, onCreated }: NewUserDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [user, setUser] = useState<User>({ ...defaultUser });
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (open) {
      setUser({ ...defaultUser });
    }
  }, [open]);

  const handleSubmit = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }

    let newUser = { ...user, nick: user.email.split("@")[0] };

    setLoading(true);
    dispatch(upsertUser({ user: newUser, password: user.password }))
      .then((r) => {
        onCreated(r);
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <DraggableDialog
      onAccept={handleSubmit}
      loading={loading}
      title={t("user.new")}
      showActions
      showCancel
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <form ref={formRef}>
          <Stack spacing={2}>
            <SettingForm title={t("user.email")} lgWidth={12}>
              <DenseFilledTextField
                fullWidth
                required
                type="email"
                value={user.email}
                onChange={(e) => setUser({ ...user, email: e.target.value })}
              />
            </SettingForm>
            <SettingForm title={t("user.password")} lgWidth={12}>
              <DenseFilledTextField
                fullWidth
                required
                slotProps={{
                  htmlInput: {
                    minLength: 6,
                  },
                }}
                type="password"
                value={user.password}
                onChange={(e) => setUser({ ...user, password: e.target.value })}
              />
            </SettingForm>
            <SettingForm title={t("user.group")} lgWidth={12}>
              <GroupSelectionInput
                required
                value={user.group_users?.toString() ?? ""}
                onChange={(e) => setUser({ ...user, group_users: parseInt(e) })}
                fullWidth
              />
            </SettingForm>
          </Stack>
        </form>
      </DialogContent>
    </DraggableDialog>
  );
};

export default NewUserDialog;
