import { DialogContent, FormControl, Stack } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { upsertGroup } from "../../../api/api";
import { GroupEnt } from "../../../api/dashboard";
import { GroupPermission } from "../../../api/user";
import { useAppDispatch } from "../../../redux/hooks";
import Boolset from "../../../util/boolset";
import { DenseFilledTextField } from "../../Common/StyledComponents";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import SettingForm from "../../Pages/Setting/SettingForm";
import GroupSelectionInput from "../Common/GroupSelectionInput";
import { NoMarginHelperText } from "../Settings/Settings";

export interface NewGroupDialogProps {
  open: boolean;
  onClose: () => void;
}

const defaultGroupBs = new Boolset("");
defaultGroupBs.sets({
  [GroupPermission.share]: true,
  [GroupPermission.share_download]: true,
  [GroupPermission.set_anonymous_permission]: true,
});
const defaultGroup: GroupEnt = {
  name: "",
  permissions: defaultGroupBs.toString(),
  max_storage: 1024 << 20, // 1GB
  settings: {
    compress_size: 1024 << 20, // 1MB
    decompress_size: 1024 << 20, // 1MB
    max_walked_files: 100000,
    trash_retention: 7 * 24 * 3600,
    source_batch: 10,
    aria2_batch: 1,
    redirected_source: true,
  },
  edges: {
    storage_policies: { id: 1 },
  },
  id: 0,
};

const NewGroupDialog = ({ open, onClose }: NewGroupDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [copyFrom, setCopyFrom] = useState<string>("0");
  const [loading, setLoading] = useState(false);
  const [group, setGroup] = useState<GroupEnt>({ ...defaultGroup });
  const formRef = useRef<HTMLFormElement>(null);
  const copyFromSrc = useRef<GroupEnt | undefined>(undefined);

  useEffect(() => {
    if (open) {
      setGroup({ ...defaultGroup });
      setCopyFrom("0");
      copyFromSrc.current = undefined;
    }
  }, [open]);

  const handleSubmit = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }

    let newGroup = { ...group };
    if (copyFrom != "0" && copyFromSrc.current) {
      newGroup = { ...copyFromSrc.current, id: 0, name: group.name };
    }

    setLoading(true);
    dispatch(upsertGroup({ group: newGroup }))
      .then((r) => {
        navigate(`/admin/group/${r.id}`);
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
      title={t("group.new")}
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
            <SettingForm title={t("group.nameOfGroup")} lgWidth={12}>
              <DenseFilledTextField
                fullWidth
                required
                value={group.name}
                onChange={(e) => setGroup({ ...group, name: e.target.value })}
              />
              <NoMarginHelperText>{t("group.nameOfGroupDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm title={t("group.copyFromExisting")} lgWidth={12}>
              <FormControl fullWidth>
                <GroupSelectionInput
                  value={copyFrom}
                  onChange={setCopyFrom}
                  onChangeGroup={(g) => {
                    copyFromSrc.current = g;
                  }}
                  emptyValue={"0"}
                  emptyText={"group.notCopy"}
                />
              </FormControl>
            </SettingForm>
          </Stack>
        </form>
      </DialogContent>
    </DraggableDialog>
  );
};

export default NewGroupDialog;
