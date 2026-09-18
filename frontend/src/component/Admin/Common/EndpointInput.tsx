import { TextFieldProps } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DenseFilledTextField } from "../../Common/StyledComponents";

export interface EndpointInputProps extends TextFieldProps<"outlined"> {
  enforceProtocol?: boolean;
  enforcePrefix?: boolean;
}

export const EndpointInput = (props: EndpointInputProps) => {
  const { t } = useTranslation("dashboard");
  const { enforceProtocol, enforcePrefix = true, onChange, ...rest } = props;
  const [value, setValue] = useState<string>((props.value as string) ?? "");
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setValue(e.target.value);
      onChange?.(e);
    },
    [onChange],
  );

  const showError = useMemo(() => {
    if (!enforceProtocol) return false;
    return value.startsWith("http://") && window.location.protocol == "https:";
  }, [enforceProtocol, value]);

  return (
    <DenseFilledTextField
      onChange={handleChange}
      value={props.value}
      slotProps={{
        htmlInput: {
          // start with http:// or https://
          pattern: enforcePrefix ? `^https?://.*$` : undefined,
          title: enforcePrefix ? t("settings.startWithProtocol") : undefined,
        },
      }}
      error={showError}
      helperText={showError ? t("settings.tlsWarning") : undefined}
      {...rest}
    />
  );
};
