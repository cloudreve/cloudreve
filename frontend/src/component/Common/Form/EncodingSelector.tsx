import {
  FormControl,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  SelectProps,
  styled,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { useTranslation } from "react-i18next";
import { NoLabelFilledSelect } from "../../FileManager/Sidebar/CustomProps/MultiSelectPropsContent.tsx";
import Translate from "../../Icons/Translate.tsx";

const encodings = [
  "ibm866",
  "iso8859_2",
  "iso8859_3",
  "iso8859_4",
  "iso8859_5",
  "iso8859_6",
  "iso8859_7",
  "iso8859_8",
  "iso8859_8I",
  "iso8859_10",
  "iso8859_13",
  "iso8859_14",
  "iso8859_15",
  "iso8859_16",
  "koi8r",
  "koi8u",
  "macintosh",
  "windows874",
  "windows1250",
  "windows1251",
  "windows1252",
  "windows1253",
  "windows1254",
  "windows1255",
  "windows1256",
  "windows1257",
  "windows1258",
  "macintoshcyrillic",
  "gbk",
  "gb18030",
  "big5",
  "eucjp",
  "iso2022jp",
  "shiftjis",
  "euckr",
  "utf16be",
  "utf16le",
];

const defaultEncodingValue = " ";

export interface EncodingSelectorProps {
  value: string;
  onChange: (value: string) => void;
  label?: string;
  size?: "small" | "medium";
  variant?: "outlined" | "standard" | "filled";
  fullWidth?: boolean;
  showIcon?: boolean;
  SelectProps?: Partial<SelectProps>;
}

export const StyledInputAdornment = styled(InputAdornment)(({ theme }) => ({
  "&.MuiInputAdornment-positionStart": {
    marginTop: "0!important",
  },
}));

const EncodingSelector = ({
  value,
  onChange,
  label,
  size = "medium",
  variant = "outlined",
  fullWidth = false,
  showIcon = true,
  SelectProps,
}: EncodingSelectorProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const displayLabel = label || t("modals.selectEncoding");

  const SelectComponent = size == "small" ? NoLabelFilledSelect : Select;
  const InputAdornmentComponent = size == "small" ? StyledInputAdornment : InputAdornment;

  return (
    <FormControl variant={variant} fullWidth={fullWidth} size={size}>
      {size != "small" && <InputLabel>{displayLabel}</InputLabel>}
      <SelectComponent
        variant={variant}
        size={size}
        startAdornment={
          showIcon &&
          !isMobile && (
            <InputAdornmentComponent position="start" sx={{ mt: 0 }}>
              <Translate />
            </InputAdornmentComponent>
          )
        }
        label={displayLabel}
        value={value}
        onChange={(e) => onChange(e.target.value as string)}
        {...SelectProps}
      >
        <MenuItem value={defaultEncodingValue}>
          <em>{t("modals.defaultEncoding")}</em>
        </MenuItem>
        {encodings.map((enc) => (
          <MenuItem key={enc} value={enc}>
            {enc}
          </MenuItem>
        ))}
      </SelectComponent>
    </FormControl>
  );
};

export { defaultEncodingValue };
export default EncodingSelector;
