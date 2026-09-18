import { FormControl, ListItemIcon, ListItemText } from "@mui/material";
import { useTranslation } from "react-i18next";
import { SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";
import { FileType } from "../../../../api/explorer.ts";
import Document from "../../../Icons/Document.tsx";
import Folder from "../../../Icons/Folder.tsx";
import { Condition } from "./ConditionBox.tsx";
import { DenseSelect } from "../../../Common/StyledComponents.tsx";

export const FileTypeCondition = ({
  condition,
  onChange,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
}) => {
  const { t } = useTranslation();
  return (
    <FormControl variant="outlined" fullWidth>
      <DenseSelect
        variant="outlined"
        value={condition.file_type ?? 0}
        onChange={(e) =>
          onChange({
            ...condition,
            file_type: e.target.value as number,
          })
        }
      >
        <SquareMenuItem value={FileType.file}>
          <ListItemIcon>
            <Document fontSize="small" />
          </ListItemIcon>
          <ListItemText
            slotProps={{
              primary: { variant: "body2" },
            }}
          >
            {t("application:fileManager.file")}
          </ListItemText>
        </SquareMenuItem>
        <SquareMenuItem value={FileType.folder}>
          <ListItemIcon>
            <Folder fontSize="small" />
          </ListItemIcon>
          <ListItemText
            slotProps={{
              primary: { variant: "body2" },
            }}
          >
            {t("application:fileManager.folder")}
          </ListItemText>
        </SquareMenuItem>
      </DenseSelect>
    </FormControl>
  );
};
