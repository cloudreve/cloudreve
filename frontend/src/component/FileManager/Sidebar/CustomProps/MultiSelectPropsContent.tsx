import { Box, Chip, MenuItem, Select, styled, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { PropsContentProps } from "./CustomPropsItem.tsx";

export const NoLabelFilledSelect = styled(Select)<{ fullSize?: boolean }>(({ theme, fullSize }) => ({
  "& .MuiSelect-select": {
    paddingTop: theme.spacing(1),
    paddingBottom: theme.spacing(1),
    ...(!fullSize
      ? {
          paddingLeft: 0,
          paddingRight: 0,
          paddingTop: theme.spacing(0.5),
          paddingBottom: theme.spacing(0.5),
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

const MultiSelectPropsContent = ({ prop, onChange, loading, readOnly, fullSize }: PropsContentProps) => {
  const { t } = useTranslation();
  const [values, setValues] = useState<string[]>(() => {
    if (prop.value) {
      try {
        return JSON.parse(prop.value);
      } catch {
        return [];
      }
    }
    return [];
  });

  useEffect(() => {
    if (prop.value) {
      try {
        setValues(JSON.parse(prop.value));
      } catch {
        setValues([]);
      }
    } else {
      setValues([]);
    }
  }, [prop.value]);

  const handleChange = (selectedValues: string[]) => {
    setValues(selectedValues);
    const newValue = JSON.stringify(selectedValues);
    if (newValue !== prop.value) {
      onChange(newValue);
    }
  };

  const handleDelete = (valueToDelete: string) => {
    const newValues = values.filter((value) => value !== valueToDelete);
    handleChange(newValues);
  };

  if (readOnly) {
    return (
      <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
        {values.length > 0
          ? values.map((value, index) => <Chip key={index} label={value} size="small" variant="outlined" />)
          : ""}
      </Box>
    );
  }

  const options = prop.props.options || [];

  return (
    <Box>
      <NoLabelFilledSelect
        variant="filled"
        fullWidth
        disabled={loading}
        value={values}
        onChange={(e) => handleChange(e.target.value as string[])}
        onClick={(e) => e.stopPropagation()}
        multiple
        displayEmpty
        fullSize={fullSize}
        renderValue={(selected) => {
          const selectedArray = Array.isArray(selected) ? selected : [];
          if (selectedArray.length === 0) {
            return (
              <Typography variant="body2" color="text.secondary">
                {t("application:fileManager.clickToEditSelect")}
              </Typography>
            );
          }
          return (
            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5 }}>
              {selectedArray.map((value, index) => (
                <Chip
                  key={index}
                  label={value}
                  size="small"
                  onDelete={() => handleDelete(value)}
                  onMouseDown={(e) => e.stopPropagation()}
                />
              ))}
            </Box>
          );
        }}
      >
        {options.map((option) => (
          <MenuItem key={option} value={option} dense>
            {option}
          </MenuItem>
        ))}
      </NoLabelFilledSelect>
    </Box>
  );
};

export default MultiSelectPropsContent;
