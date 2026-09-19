import { useTheme } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getShareDetail, getShareList } from "../../../api/api.ts";
import { Share } from "../../../api/dashboard.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { DenseAutocomplete, DenseFilledTextField } from "../../Common/StyledComponents.tsx";

export interface SharesInputProps {
  value: number[];
  onChange: (value: number[]) => void;
}

const optionLabel = (s: Share) => (s.edges?.file?.name ? `${s.edges.file.name} (#${s.id})` : `#${s.id}`);

const SharesInput = ({ value, onChange }: SharesInputProps) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [options, setOptions] = useState<Share[]>([]);
  const [selected, setSelected] = useState<Share[]>([]);

  // Resolve currently selected share ids to share objects for display.
  useEffect(() => {
    let mounted = true;
    Promise.all(value.map((id) => dispatch(getShareDetail(id)).catch(() => null)))
      .then((res) => {
        if (mounted) {
          setSelected(res.filter((s): s is Share => s != null));
        }
      });
    return () => {
      mounted = false;
    };
  }, [value]);

  // Load the most recent shares as the option pool; filtering is client-side.
  useEffect(() => {
    dispatch(
      getShareList({
        page: 1,
        page_size: 50,
        order_by: "id",
        order_direction: "desc",
      }),
    )
      .then((res) => setOptions(res.shares))
      .catch(() => {});
  }, []);

  return (
    <DenseAutocomplete
      multiple
      options={options}
      value={selected}
      isOptionEqualToValue={(o, v) => (o as Share).id === (v as Share).id}
      getOptionLabel={(o) => optionLabel(o as Share)}
      onChange={(_, v) => onChange((v as Share[]).map((s) => s.id))}
      renderInput={(params) => (
        <DenseFilledTextField
          {...params}
          sx={{
            "& .MuiInputBase-root.MuiOutlinedInput-root": {
              paddingTop: theme.spacing(0.6),
              paddingBottom: theme.spacing(0.6),
            },
            mt: 0,
          }}
          variant="outlined"
          margin="dense"
          placeholder={t("dashboard:settings.searchShare")}
          type="text"
          fullWidth
        />
      )}
    />
  );
};

export default SharesInput;
