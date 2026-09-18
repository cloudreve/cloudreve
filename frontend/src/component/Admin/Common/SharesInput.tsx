import { Box, debounce, useTheme } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getShareList } from "../../../api/api.ts";
import { Share } from "../../../api/dashboard.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { DenseAutocomplete, DenseFilledTextField, NoWrapBox, SquareChip } from "../../Common/StyledComponents.tsx";
import FileTypeIcon from "../../FileManager/Explorer/FileTypeIcon.tsx";
import LinkDismiss from "../../Icons/LinkDismiss.tsx";

export interface SharesInputProps {}

const SharesInput = (props: SharesInputProps) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const [options, setOptions] = useState<number[]>([]);

  return (
    <DenseAutocomplete
      multiple
      options={options}
      blurOnSelect
      renderInput={(params) => (
        <DenseFilledTextField
          {...params}
          sx={{
            "& .MuiInputBase-root": {},
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
