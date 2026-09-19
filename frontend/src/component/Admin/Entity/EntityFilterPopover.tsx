import { Box, Button, ListItemText, Popover, PopoverProps, Stack } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { EntityType } from "../../../api/explorer";
import { DenseFilledTextField, DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import { EntityTypeText } from "../../FileManager/Sidebar/Data";
import SettingForm from "../../Pages/Setting/SettingForm";
import SinglePolicySelectionInput from "../Common/SinglePolicySelectionInput";
export interface EntityFilterPopoverProps extends PopoverProps {
  storagePolicy: string;
  setStoragePolicy: (storagePolicy: string) => void;
  owner: string;
  setOwner: (owner: string) => void;
  type?: EntityType;
  setType: (type?: EntityType) => void;
  refCount: string;
  setRefCount: (refCount: string) => void;
  clearFilters: () => void;
}

const EntityFilterPopover = ({
  storagePolicy,
  setStoragePolicy,
  owner,
  setOwner,
  type,
  setType,
  refCount,
  setRefCount,
  clearFilters,
  onClose,
  open,
  ...rest
}: EntityFilterPopoverProps) => {
  const { t } = useTranslation("dashboard");

  // Create local state to track changes before applying
  const [localStoragePolicy, setLocalStoragePolicy] = useState(storagePolicy);
  const [localOwner, setLocalOwner] = useState(owner);
  const [localType, setLocalType] = useState(type);
  const [localRefCountOp, setLocalRefCountOp] = useState("");
  const [localRefCountValue, setLocalRefCountValue] = useState("");

  // Initialize local state when popup opens
  useEffect(() => {
    if (open) {
      setLocalStoragePolicy(storagePolicy);
      setLocalOwner(owner);
      setLocalType(type);
      const [op, value] = refCount.split(":");
      setLocalRefCountOp(refCount ? op : "");
      setLocalRefCountValue(value ?? "");
    }
  }, [open]);

  // Apply filters and close popover
  const handleApplyFilters = () => {
    setStoragePolicy(localStoragePolicy);
    setOwner(localOwner);
    setType(localType);
    setRefCount(
      localRefCountOp == "" ? "" : localRefCountOp == "stale" ? "stale" : `${localRefCountOp}:${localRefCountValue || "0"}`,
    );
    onClose?.({}, "backdropClick");
  };

  // Reset filters and close popover
  const handleResetFilters = () => {
    setLocalStoragePolicy("");
    setLocalOwner("");
    setLocalType(undefined);
    setLocalRefCountOp("");
    setLocalRefCountValue("");
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
        <SettingForm title={t("file.uploaderID")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localOwner}
            onChange={(e) => setLocalOwner(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <SettingForm title={t("file.blobType")} noContainer lgWidth={12}>
          <DenseSelect
            fullWidth
            displayEmpty
            value={localType != undefined ? localType : -1}
            onChange={(e) => setLocalType(e.target.value === -1 ? undefined : (e.target.value as EntityType))}
          >
            {[EntityType.version, EntityType.thumbnail, EntityType.live_photo].map((type) => (
              <SquareMenuItem key={type} value={type}>
                <ListItemText
                  primary={t(EntityTypeText[type])}
                  slotProps={{
                    primary: {
                      variant: "body2",
                    },
                  }}
                />
              </SquareMenuItem>
            ))}
            <SquareMenuItem value={-1}>
              <ListItemText
                primary={<em>{t("user.all")}</em>}
                slotProps={{
                  primary: {
                    variant: "body2",
                  },
                }}
              />
            </SquareMenuItem>
          </DenseSelect>
        </SettingForm>

        <SettingForm title={t("file.referenceCount")} noContainer lgWidth={12}>
          <Stack direction="row" spacing={1}>
            <DenseSelect
              displayEmpty
              value={localRefCountOp}
              onChange={(e) => setLocalRefCountOp(e.target.value)}
              sx={{ minWidth: 140 }}
            >
              <SquareMenuItem value="">
                <ListItemText primary={<em>{t("user.all")}</em>} slotProps={{ primary: { variant: "body2" } }} />
              </SquareMenuItem>
              <SquareMenuItem value="stale">
                <ListItemText primary={t("file.awaitingRecycle")} slotProps={{ primary: { variant: "body2" } }} />
              </SquareMenuItem>
              <SquareMenuItem value="gt">
                <ListItemText primary={t("file.refCountGt")} slotProps={{ primary: { variant: "body2" } }} />
              </SquareMenuItem>
              <SquareMenuItem value="lt">
                <ListItemText primary={t("file.refCountLt")} slotProps={{ primary: { variant: "body2" } }} />
              </SquareMenuItem>
              <SquareMenuItem value="eq">
                <ListItemText primary={t("file.refCountEq")} slotProps={{ primary: { variant: "body2" } }} />
              </SquareMenuItem>
            </DenseSelect>
            {localRefCountOp != "" && localRefCountOp != "stale" && (
              <DenseFilledTextField
                value={localRefCountValue}
                onChange={(e) => setLocalRefCountValue(e.target.value.replace(/\D/g, ""))}
                placeholder="0"
                size="small"
                inputProps={{ inputMode: "numeric" }}
                sx={{ width: 80 }}
              />
            )}
          </Stack>
        </SettingForm>

        <SettingForm title={t("file.storagePolicy")} noContainer lgWidth={12}>
          <SinglePolicySelectionInput
            value={localStoragePolicy == "" ? -1 : parseInt(localStoragePolicy)}
            onChange={(value) => setLocalStoragePolicy(value.toString())}
            emptyValue={-1}
            emptyText={t("user.all")}
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

export default EntityFilterPopover;
