import { Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { NoLabelFilledTextField } from "../../../Common/StyledComponents.tsx";
import { PropsContentProps } from "./CustomPropsItem.tsx";

const NumberPropsContent = ({ prop, onChange, loading, readOnly, fullSize }: PropsContentProps) => {
  const { t } = useTranslation();
  const [value, setValue] = useState(prop.value);

  useEffect(() => {
    setValue(prop.value);
  }, [prop.value]);

  const onBlur = () => {
    if (value !== prop.value) {
      onChange(value);
    }
  };

  if (readOnly) {
    return <Typography variant="body2">{value}</Typography>;
  }

  return (
    <NoLabelFilledTextField
      type="number"
      variant="filled"
      placeholder={t("application:fileManager.clickToEdit")}
      fullWidth
      disabled={loading}
      slotProps={{
        input: {
          inputProps: {
            max: prop.props.max,
            min: prop.props.min ?? 0,
          },
        },
      }}
      onClick={(e) => e.stopPropagation()}
      onChange={(e) => setValue(e.target.value)}
      value={value ?? ""}
      onBlur={onBlur}
      fullSize={fullSize}
      required
    />
  );
};

export default NumberPropsContent;
