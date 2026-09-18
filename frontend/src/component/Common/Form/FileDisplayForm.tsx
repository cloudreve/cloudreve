import FileBadge from "../../FileManager/FileBadge.tsx";
import { FileResponse } from "../../../api/explorer.ts";
import { StyledTextField } from "./PathSelectorForm.tsx";

export interface FileDisplayFormProps {
  file: FileResponse;
  label: string;
}

export const FileDisplayForm = ({ file, label }: FileDisplayFormProps) => {
  return (
    <StyledTextField
      variant="outlined"
      InputProps={{
        readOnly: true,
        startAdornment: <FileBadge file={file} clickable={false} />,
      }}
      label={label}
      fullWidth
    />
  );
};
