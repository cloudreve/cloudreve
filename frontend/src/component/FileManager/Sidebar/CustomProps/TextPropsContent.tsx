import { Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { NoLabelFilledTextField } from "../../../Common/StyledComponents.tsx";
import { PropsContentProps } from "./CustomPropsItem.tsx";

const TextPropsContent = ({ prop, onChange, loading, readOnly, fullSize }: PropsContentProps) => {
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
      variant="filled"
      fullSize={fullSize}
      placeholder={t("application:fileManager.clickToEdit")}
      disabled={loading}
      fullWidth
      slotProps={{
        input: {
          inputProps: {
            maxLength: prop.props.max,
          },
        },
      }}
      onClick={(e) => e.stopPropagation()}
      onChange={(e) => setValue(e.target.value)}
      value={value ?? ""}
      onBlur={onBlur}
      required
      multiline
    />
  );
};

export default TextPropsContent;
