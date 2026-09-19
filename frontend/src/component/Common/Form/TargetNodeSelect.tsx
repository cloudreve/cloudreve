import { FormControl, InputLabel, MenuItem, Select } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../redux/hooks.ts";

export interface TargetNodeSelectProps {
  value: string;
  onChange: (value: string) => void;
}

export const useShowTargetNodeSelect = () =>
  useAppSelector(
    (state) =>
      !!state.siteConfig.explorer?.config?.allow_select_node &&
      (state.siteConfig.explorer?.config?.task_nodes?.length ?? 0) > 0,
  );

const TargetNodeSelect = ({ value, onChange }: TargetNodeSelectProps) => {
  const { t } = useTranslation();
  const show = useShowTargetNodeSelect();
  const taskNodes = useAppSelector((state) => state.siteConfig.explorer?.config?.task_nodes);

  if (!show) {
    return null;
  }

  return (
    <FormControl variant="outlined" fullWidth>
      <InputLabel>{t("application:modals.processNode")}</InputLabel>
      <Select
        variant="outlined"
        label={t("application:modals.processNode")}
        value={value}
        onChange={(e) => onChange(e.target.value as string)}
      >
        <MenuItem value="">
          <em>{t("application:modals.remoteDownloadNodeAuto")}</em>
        </MenuItem>
        {(taskNodes ?? []).map((n) => (
          <MenuItem key={n.id} value={n.id}>
            {n.name}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
};

export default TargetNodeSelect;
