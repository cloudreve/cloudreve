import { useTranslation } from "react-i18next";
import { useMemo } from "react";
import { Box, Collapse, Divider, IconButton, InputAdornment, Stack } from "@mui/material";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import FormControl from "@mui/material/FormControl";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Add from "../../../Icons/Add.tsx";
import { TransitionGroup } from "react-transition-group";
import { NoMarginHelperText, StyledInputAdornment } from "../Settings.tsx";

export interface SiteURLInputProps {
  urls: string;
  onChange: (url: string) => void;
}

const SiteURLInput = ({ urls, onChange }: SiteURLInputProps) => {
  const { t } = useTranslation("dashboard");
  const urlSplit = useMemo(() => {
    return urls.split(",").map((url) => url);
  }, [urls]);

  const onUrlChange = (index: number) => (e: React.ChangeEvent<HTMLInputElement>) => {
    const newUrls = [...urlSplit];
    newUrls[index] = e.target.value;
    onChange(newUrls.join(","));
  };

  const removeUrl = (index: number) => () => {
    const newUrls = [...urlSplit];
    newUrls.splice(index, 1);
    onChange(newUrls.join(","));
  };

  return (
    <Stack spacing={1}>
      <FormControl fullWidth>
        <DenseFilledTextField
          fullWidth
          onChange={onUrlChange(0)}
          value={urlSplit[0]}
          InputProps={{
            startAdornment: (
              <StyledInputAdornment disableTypography position="start">
                {t("settings.primarySiteURL")}
              </StyledInputAdornment>
            ),
          }}
          required
        />
        <NoMarginHelperText>{t("settings.primarySiteURLDes")}</NoMarginHelperText>
      </FormControl>
      <Divider />
      <NoMarginHelperText>{t("settings.secondaryDes")}</NoMarginHelperText>
      <TransitionGroup>
        {urlSplit.slice(1).map((url, index) => (
          <Collapse key={index}>
            <FormControl fullWidth sx={{ mb: 1 }}>
              <DenseFilledTextField
                fullWidth
                onChange={onUrlChange(index + 1)}
                value={url}
                InputProps={{
                  startAdornment: (
                    <StyledInputAdornment disableTypography position="start">
                      {t("settings.secondarySiteURL")}
                    </StyledInputAdornment>
                  ),
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton size={"small"} onClick={removeUrl(index + 1)}>
                        <Dismiss fontSize={"small"} />
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
                required
              />
            </FormControl>
          </Collapse>
        ))}
      </TransitionGroup>
      <Box sx={{ mt: "0!important" }}>
        <SecondaryButton variant={"contained"} startIcon={<Add />} onClick={() => onChange(`${urls},`)}>
          {t("settings.addSecondary")}
        </SecondaryButton>
      </Box>
    </Stack>
  );
};

export default SiteURLInput;
