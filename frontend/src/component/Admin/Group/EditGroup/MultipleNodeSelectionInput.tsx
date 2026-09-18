import { ListItemText } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../../redux/hooks";
import { DenseSelect } from "../../../Common/StyledComponents";

const MultipleNodeSelectionInput = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();

  return (
    <DenseSelect
      multiple
      displayEmpty
      sx={{
        minHeight: 39,
      }}
      MenuProps={{
        PaperProps: { sx: { maxWidth: 230 } },
        MenuListProps: {
          sx: {
            "& .MuiMenuItem-root": {
              whiteSpace: "normal",
            },
          },
        },
      }}
      renderValue={(selected) => {
        return (
          <ListItemText
            primary={<em>{t("group.allNodes")}</em>}
            slotProps={{
              primary: { color: "textSecondary", variant: "body2" },
            }}
          />
        );
      }}
    ></DenseSelect>
  );
};

export default MultipleNodeSelectionInput;
