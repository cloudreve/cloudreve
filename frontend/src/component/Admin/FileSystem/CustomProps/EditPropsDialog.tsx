import { Box, DialogContent, FormControl, Grid2, Link } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { CustomProps } from "../../../../api/explorer";
import { DenseFilledTextField } from "../../../Common/StyledComponents";
import DraggableDialog from "../../../Dialogs/DraggableDialog";
import { getPropsContent } from "../../../FileManager/Sidebar/CustomProps/CustomPropsItem";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText } from "../../Settings/Settings";
import { FieldTypes } from "./DraggableCustomPropsRow";

interface EditPropsDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (props: CustomProps) => void;
  isNew: boolean;
  props?: CustomProps;
}

const EditPropsDialog = ({ open, onClose, onSave, isNew, props }: EditPropsDialogProps) => {
  const { t } = useTranslation("dashboard");
  const formRef = useRef<HTMLFormElement>(null);
  const [editProps, setEditProps] = useState<CustomProps | undefined>(props);

  const handleSave = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }

    onSave({ ...editProps } as CustomProps);
    onClose();
  };

  useEffect(() => {
    if (props) {
      setEditProps({ ...props });
    }

    if (!open) {
      setTimeout(() => {
        setEditProps(undefined);
      }, 100);
    }
  }, [open, props]);

  if (!editProps || !editProps.type) return null;

  const fieldType = FieldTypes[editProps?.type];
  return (
    <DraggableDialog
      title={isNew ? t("customProps.addProp") : t("customProps.editProp")}
      showActions
      showCancel
      onAccept={handleSave}
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <Box component={"form"} ref={formRef} sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
          <SettingForm title={t("customProps.id")} lgWidth={12}>
            <FormControl fullWidth>
              <DenseFilledTextField
                disabled={!isNew}
                slotProps={{
                  htmlInput: {
                    max: 128,
                    pattern: "^[a-zA-Z0-9_-]+$",
                    title: t("customProps.idPatternDes"),
                  },
                }}
                value={editProps?.id || ""}
                required
                onChange={(e) => setEditProps({ ...editProps, id: e.target.value } as CustomProps)}
              />
              <NoMarginHelperText>{t("customProps.idDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
          <SettingForm title={t("settings.displayName")} lgWidth={12}>
            <FormControl fullWidth>
              <DenseFilledTextField
                value={editProps?.name || ""}
                onChange={(e) =>
                  setEditProps({
                    ...editProps,
                    name: e.target.value,
                  } as CustomProps)
                }
                required
              />
              <NoMarginHelperText>{t("settings.displayNameDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
          <SettingForm title={t("customProps.icon")} lgWidth={12}>
            <FormControl fullWidth>
              <DenseFilledTextField
                value={editProps?.icon || ""}
                onChange={(e) => setEditProps({ ...editProps, icon: e.target.value } as CustomProps)}
              />
              <NoMarginHelperText>
                {
                  <Trans
                    i18nKey="dashboard:customProps.iconDes"
                    components={[<Link target="_blank" href="https://icon-sets.iconify.design/" />]}
                  />
                }
              </NoMarginHelperText>
            </FormControl>
          </SettingForm>
          {(fieldType.minTitle || fieldType.maxTitle) && (
            <Grid2 container spacing={2} size={{ xs: 12 }}>
              {fieldType.minTitle && (
                <SettingForm title={t(fieldType.minTitle)} lgWidth={6} noContainer>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      type="number"
                      value={editProps?.min || ""}
                      onChange={(e) => setEditProps({ ...editProps, min: e.target.value === "" ? undefined : parseInt(e.target.value) } as CustomProps)}
                    />
                    {fieldType.minDes && <NoMarginHelperText>{t(fieldType.minDes)}</NoMarginHelperText>}
                  </FormControl>
                </SettingForm>
              )}
              {fieldType.maxTitle && (
                <SettingForm title={t(fieldType.maxTitle)} lgWidth={6} noContainer>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      type="number"
                      required={fieldType.maxRequired}
                      value={editProps?.max || ""}
                      onChange={(e) => setEditProps({ ...editProps, max: e.target.value === "" ? undefined : parseInt(e.target.value) } as CustomProps)}
                    />
                    {fieldType.maxDes && <NoMarginHelperText>{t(fieldType.maxDes)}</NoMarginHelperText>}
                  </FormControl>
                </SettingForm>
              )}
            </Grid2>
          )}
          {fieldType.showOptions && (
            <SettingForm title={t("customProps.options")} lgWidth={12}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  multiline
                  rows={4}
                  value={editProps?.options?.join("\n") || ""}
                  onChange={(e) => setEditProps({ ...editProps, options: e.target.value.split("\n") } as CustomProps)}
                />
                <NoMarginHelperText>{t("customProps.optionsDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          )}
          <SettingForm title={t("customProps.default")} lgWidth={12}>
            <FormControl fullWidth>
              {getPropsContent(
                {
                  props: editProps,
                  id: editProps.id,
                  value: editProps.default ?? "",
                },
                (value) => {
                  setEditProps({ ...editProps, default: value } as CustomProps);
                },
                false,
                false,
                true,
              )}
            </FormControl>
          </SettingForm>
        </Box>
      </DialogContent>
    </DraggableDialog>
  );
};

export default EditPropsDialog;
