import { styled, TextField, TextFieldProps } from "@mui/material";
import { useCallback } from "react";
import { FileType } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { selectPath } from "../../../redux/thunks/dialog.ts";
import FileBadge from "../../FileManager/FileBadge.tsx";

export interface PathSelectorFormProps {
  path: string;
  label?: string;
  onChange: (path: string) => void;
  variant?: string;
  textFieldProps?: TextFieldProps;
  allowedFs?: string[];
}

export const StyledTextField = styled(TextField)({
  "& .MuiInputBase-root": {
    paddingLeft: "8px",
    cursor: "pointer",
  },
  "& .MuiOutlinedInput-input": {
    paddingLeft: "8px",
    cursor: "pointer",
  },
});

export const PathSelectorForm = ({ path, onChange, label, variant, textFieldProps }: PathSelectorFormProps) => {
  const dispatch = useAppDispatch();
  const onClick = useCallback(() => {
    dispatch(selectPath(variant ?? "saveTo", path)).then((path) => {
      onChange(path);
    });
  }, [dispatch]);
  return (
    <StyledTextField
      onClick={onClick}
      variant="outlined"
      InputProps={{
        readOnly: true,
        startAdornment: <FileBadge simplifiedFile={{ path, type: FileType.folder }} clickable={false} />,
      }}
      label={label}
      fullWidth
      {...textFieldProps}
    />
  );
};
