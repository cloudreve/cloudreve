import { MenuItem, Select, styled, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { PropsContentProps } from "./CustomPropsItem.tsx";

const NoLabelFilledSelect = styled(Select)<{ fullSize?: boolean }>(({ theme, fullSize }) => ({
  "& .MuiSelect-select": {
    paddingTop: theme.spacing(1),
    paddingBottom: theme.spacing(1),
    ...(!fullSize
      ? {
          paddingLeft: 0,
          paddingRight: 0,
          paddingTop: 0,
          paddingBottom: 0,
        }
      : {}),
    fontSize: theme.typography.body2.fontSize,
    "&.Mui-disabled": {
      borderBottomStyle: "none",
      "&::before": {
        borderBottomStyle: "none !important",
      },
    },
  },
  "&.MuiInputBase-root.MuiFilledInput-root.MuiSelect-root": {
    ...(!fullSize
      ? {
          borderRadius: 0,
          backgroundColor: "initial",
        }
      : {}),
    "&.Mui-disabled": {
      borderBottomStyle: "none",
      "&::before": {
        borderBottomStyle: "none !important",
      },
    },
  },
}));

const SelectPropsContent = ({ prop, onChange, loading, readOnly, fullSize }: PropsContentProps) => {
  const { t } = useTranslation();
  const [value, setValue] = useState(prop.value || "");

  useEffect(() => {
    setValue(prop.value || "");
  }, [prop.value]);

  const handleChange = (selectedValue: string) => {
    setValue(selectedValue);
    if (selectedValue !== prop.value) {
      onChange(selectedValue);
    }
  };

  if (readOnly) {
    return <Typography variant="body2">{value}</Typography>;
  }

  const options = prop.props.options || [];

  return (
    <NoLabelFilledSelect
      variant="filled"
      fullWidth
      fullSize={fullSize}
      disabled={loading}
      value={value}
      onChange={(e) => handleChange(e.target.value as string)}
      onClick={(e) => e.stopPropagation()}
      displayEmpty
      renderValue={(selected) => {
        if (!selected) {
          return (
            <Typography variant="body2" color="text.secondary">
              {t("application:fileManager.clickToEditSelect")}
            </Typography>
          );
        }
        return selected as string;
      }}
    >
      {options.map((option) => (
        <MenuItem key={option} value={option} dense>
          {option}
        </MenuItem>
      ))}
    </NoLabelFilledSelect>
  );
};

export default SelectPropsContent;
