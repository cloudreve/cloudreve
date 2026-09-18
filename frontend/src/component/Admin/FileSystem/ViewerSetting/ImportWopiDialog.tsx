import { DialogContent, Link } from "@mui/material";
import { useCallback, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { getWopiDiscovery } from "../../../../api/api.ts";
import { ViewerGroup } from "../../../../api/explorer.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { Code } from "../../../Common/Code.tsx";
import { NoMarginHelperText } from "../../Settings/Settings.tsx";

export interface ImportWopiDialogProps {
  open: boolean;
  onClose: () => void;
  onImported: (v: ViewerGroup) => void;
}

const ImportWopiDialog = ({ open, onClose, onImported }: ImportWopiDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [endpoint, setEndpoint] = useState("");
  const [loading, setLoading] = useState(false);

  const onSubmit = useCallback(() => {
    setLoading(true);
    dispatch(
      getWopiDiscovery({
        endpoint,
      }),
    )
      .then((res) => {
        onImported(res);
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  }, [endpoint, onClose, onImported]);

  return (
    <DraggableDialog
      title={t("settings.importWopi")}
      showActions
      showCancel
      onAccept={onSubmit}
      disabled={endpoint == ""}
      loading={loading}
      dialogProps={{
        fullWidth: true,
        maxWidth: "md",
        open,
        onClose,
      }}
    >
      <DialogContent>
        <SettingForm lgWidth={12} title={t("settings.wopiEndpoint")}>
          <DenseFilledTextField value={endpoint} fullWidth onChange={(e) => setEndpoint(e.target.value)} />
          <NoMarginHelperText>
            <Trans
              ns="dashboard"
              i18nKey="settings.wopiDes"
              components={[<Code />, <Link href="https://docs.cloudreve.org/usage/wopi" target="_blank" />]}
            />
          </NoMarginHelperText>
        </SettingForm>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ImportWopiDialog;
