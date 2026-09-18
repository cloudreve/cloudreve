import {
  FilledInput,
  FilledInputProps,
  FormControl,
  FormHelperText,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  styled,
} from "@mui/material";
import { parseInt } from "lodash";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../redux/hooks.ts";
import { DenseFilledTextField } from "./StyledComponents.tsx";

const unitTransform = (v?: number): number[] => {
  if (!v || v.toString() === "0") {
    return [0, 1024 * 1024];
  }
  for (let i = 4; i >= 0; i--) {
    const base = Math.pow(1024, i);
    if (v % base === 0) {
      return [v / base, base];
    }
  }

  return [0, 1024 * 1024];
};

export interface SizeInputProps {
  onChange: (size: number) => void;
  min?: number;
  value: number;
  required?: boolean;
  label?: string;
  max?: number;
  suffix?: string;
  inputProps?: FilledInputProps;
  variant?: "filled" | "outlined";
  allowZero?: boolean;
}

export const StyledSelect = styled(Select)(() => ({
  "& .MuiFilledInput-input": {
    paddingTop: "5px",
    "&:focus": {
      backgroundColor: "initial",
    },
  },
  minWidth: "70px",
  marginTop: "14px",
  backgroundColor: "initial",
}));

export const StyleOutlinedSelect = styled(Select)(({ theme }) => ({
  "& .MuiFilledInput-input": {
    paddingTop: "5px",
    "&:focus": {
      backgroundColor: "initial",
    },
  },
  minWidth: "70px",
  backgroundColor: "initial",
  fontSize: theme.typography.body2.fontSize,
}));

export default function SizeInput({
  onChange,
  min,
  value,
  required,
  label,
  max,
  inputProps,
  allowZero = true,
  suffix,
  variant = "filled",
}: SizeInputProps) {
  const dispatch = useAppDispatch();
  const { t } = useTranslation();

  const [unit, setUnit] = useState(1);
  const [val, setVal] = useState(value);
  const [err, setError] = useState("");

  useEffect(() => {
    onChange(val * unit);
    if ((max && val * unit > max) || (min && val * unit < min)) {
      setError(t("common:incorrectSizeInput"));
    } else {
      setError("");
    }
  }, [val, unit, max, min]);

  useEffect(() => {
    const res = unitTransform(value);
    setUnit(res[1]);
    setVal(res[0]);
  }, [value]);

  if (variant === "outlined") {
    return (
      <DenseFilledTextField
        value={val}
        type={"number"}
        inputProps={{ step: 1, min: allowZero ? 0 : 1 }}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setVal(parseInt(e.target.value) ?? 0)}
        error={err !== ""}
        helperText={err}
        required={required}
        InputProps={{
          endAdornment: (
            <InputAdornment position="end">
              <StyleOutlinedSelect
                variant={"filled"}
                size={"small"}
                value={unit}
                onChange={(e) => setUnit(e.target.value as number)}
              >
                <MenuItem dense value={1}>
                  B{suffix && suffix}
                </MenuItem>
                <MenuItem dense value={1024}>
                  KB{suffix && suffix}
                </MenuItem>
                <MenuItem dense value={1024 * 1024}>
                  MB{suffix && suffix}
                </MenuItem>
                <MenuItem dense value={1024 * 1024 * 1024}>
                  GB{suffix && suffix}
                </MenuItem>
                <MenuItem dense value={1024 * 1024 * 1024 * 1024}>
                  TB{suffix && suffix}
                </MenuItem>
              </StyleOutlinedSelect>
            </InputAdornment>
          ),
          ...inputProps,
        }}
      />
    );
  }

  return (
    <FormControl variant={"filled"} error={err !== ""}>
      <InputLabel htmlFor="component-helper">{label}</InputLabel>
      <FilledInput
        value={val}
        type={"number"}
        inputProps={{ step: 1, min: allowZero ? 0 : 1 }}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setVal(parseInt(e.target.value) ?? 0)}
        required={required}
        endAdornment={
          <InputAdornment position="end">
            <StyledSelect
              variant={"filled"}
              size={"small"}
              value={unit}
              onChange={(e) => setUnit(e.target.value as number)}
            >
              <MenuItem dense value={1}>
                B{suffix && suffix}
              </MenuItem>
              <MenuItem dense value={1024}>
                KB{suffix && suffix}
              </MenuItem>
              <MenuItem dense value={1024 * 1024}>
                MB{suffix && suffix}
              </MenuItem>
              <MenuItem dense value={1024 * 1024 * 1024}>
                GB{suffix && suffix}
              </MenuItem>
              <MenuItem dense value={1024 * 1024 * 1024 * 1024}>
                TB{suffix && suffix}
              </MenuItem>
            </StyledSelect>
          </InputAdornment>
        }
        {...inputProps}
      />
      {err !== "" && <FormHelperText>{err}</FormHelperText>}
    </FormControl>
  );
}
