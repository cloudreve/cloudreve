import { ExpandMoreRounded } from "@mui/icons-material";
import { LoadingButton } from "@mui/lab";
import { AccordionDetails, Box, FormControl, FormControlLabel, InputAdornment, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import * as React from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { sendTestThumbGeneratorExecutable } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import SizeInput from "../../../Common/SizeInput.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { DenseFilledTextField, StyledCheckbox } from "../../../Common/StyledComponents.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSectionContent } from "../Settings.tsx";
import { AccordionSummary, StyledAccordion } from "../UserSession/SSOSettings.tsx";

export interface GeneratorsProps {
  values: {
    [key: string]: any;
  };
  setSetting: (v: { [key: string]: any }) => void;
}

interface GeneratorRenderProps {
  name: string;
  des: string;
  enableFlag?: string;
  executableSetting?: string;
  maxSizeSetting?: string;
  readOnly?: boolean;
  inputs?: {
    name: string;
    label: string;
    des: string;
    required?: boolean;
  }[];
}

const generators: GeneratorRenderProps[] = [
  {
    name: "policyBuiltin",
    des: "policyBuiltinDes",
    readOnly: true,
  },
  {
    name: "musicCover",
    des: "musicCoverDes",
    enableFlag: "thumb_music_cover_enabled",
    maxSizeSetting: "thumb_music_cover_max_size",
    inputs: [
      {
        name: "thumb_music_cover_exts",
        label: "generatorExts",
        des: "generatorExtsDes",
      },
    ],
  },
  {
    name: "libreOffice",
    des: "libreOfficeDes",
    enableFlag: "thumb_libreoffice_enabled",
    maxSizeSetting: "thumb_libreoffice_max_size",
    executableSetting: "thumb_libreoffice_path",
    inputs: [
      {
        name: "thumb_libreoffice_exts",
        label: "generatorExts",
        des: "generatorExtsDes",
      },
    ],
  },
  {
    name: "libraw",
    des: "librawDes",
    enableFlag: "thumb_libraw_enabled",
    maxSizeSetting: "thumb_libraw_max_size",
    executableSetting: "thumb_libraw_path",
    inputs: [
      {
        name: "thumb_libraw_exts",
        label: "generatorExts",
        des: "generatorExtsDes",
      },
    ],
  },
  {
    name: "vips",
    des: "vipsDes",
    enableFlag: "thumb_vips_enabled",
    maxSizeSetting: "thumb_vips_max_size",
    executableSetting: "thumb_vips_path",
    inputs: [
      {
        name: "thumb_vips_exts",
        label: "generatorExts",
        des: "generatorExtsDes",
      },
    ],
  },
  {
    name: "ffmpeg",
    des: "ffmpegDes",
    enableFlag: "thumb_ffmpeg_enabled",
    maxSizeSetting: "thumb_ffmpeg_max_size",
    executableSetting: "thumb_ffmpeg_path",
    inputs: [
      {
        name: "thumb_ffmpeg_exts",
        label: "generatorExts",
        des: "generatorExtsDes",
      },
      {
        name: "thumb_ffmpeg_seek",
        label: "ffmpegSeek",
        des: "ffmpegSeekDes",
        required: true,
      },
      {
        name: "thumb_ffmpeg_extra_args",
        label: "ffmpegExtraArgs",
        des: "ffmpegExtraArgsDes",
      },
    ],
  },
  {
    name: "cloudreveBuiltin",
    maxSizeSetting: "thumb_builtin_max_size",
    des: "cloudreveBuiltinDes",
    enableFlag: "thumb_builtin_enabled",
  },
];

const Generators = ({ values, setSetting }: GeneratorsProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [testing, setTesting] = useState(false);
  const { enqueueSnackbar } = useSnackbar();

  const handleEnableChange = (name: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
    setSetting({
      [name]: e.target.checked ? "1" : "0",
    });
    const newValues = { ...values, [name]: e.target.checked ? "1" : "0" };
    if (
      (newValues["thumb_libreoffice_enabled"] === "1" || newValues["thumb_music_cover_enabled"] === "1") &&
      newValues["thumb_builtin_enabled"] === "0" &&
      newValues["thumb_vips_enabled"] === "0"
    ) {
      enqueueSnackbar({
        message: t("settings.thumbDependencyWarning"),
        variant: "warning",
        action: DefaultCloseAction,
      });
    }
  };

  const doTest = (name: string, executable: string) => {
    setTesting(true);
    dispatch(
      sendTestThumbGeneratorExecutable({
        name,
        executable,
      }),
    )
      .then((res) => {
        enqueueSnackbar({
          message: t("settings.executableTestSuccess", { version: res }),
          variant: "success",
          action: DefaultCloseAction,
        });
      })
      .finally(() => {
        setTesting(false);
      });
  };

  return (
    <Box>
      {generators.map((g) => (
        <StyledAccordion key={g.name} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreRounded />}>
            <FormControlLabel
              onClick={(event) => event.stopPropagation()}
              onFocus={(event) => event.stopPropagation()}
              control={
                <StyledCheckbox
                  size={"small"}
                  checked={g.readOnly || isTrueVal(values[g.enableFlag ?? ""])}
                  onChange={handleEnableChange(g.enableFlag ?? "")}
                />
              }
              label={t(`settings.${g.name}`)}
              disabled={g.readOnly}
            />
          </AccordionSummary>
          <AccordionDetails sx={{ display: "block" }}>
            <Typography color="textSecondary" variant={"body2"}>
              {t(`settings.${g.des}`)}
            </Typography>
            <SettingSectionContent sx={{ mt: 2 }}>
              {g.executableSetting && (
                <SettingForm lgWidth={12} title={t("settings.executable")}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      required
                      value={values[g.executableSetting]}
                      InputProps={{
                        endAdornment: (
                          <InputAdornment position="end">
                            <LoadingButton
                              onClick={() => doTest(g.name, values[g.executableSetting ?? ""])}
                              loading={testing}
                              color="primary"
                            >
                              <span>{t("settings.executableTest")}</span>
                            </LoadingButton>
                          </InputAdornment>
                        ),
                      }}
                      onChange={(e) =>
                        setSetting({
                          [g.executableSetting ?? ""]: e.target.value,
                        })
                      }
                    />
                    <NoMarginHelperText>{t("settings.executableDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              )}
              {g.maxSizeSetting && (
                <SettingForm lgWidth={12} title={t("settings.thumbMaxSize")}>
                  <FormControl fullWidth>
                    <SizeInput
                      variant={"outlined"}
                      required
                      allowZero={false}
                      value={parseInt(values[g.maxSizeSetting ?? ""]) ?? 0}
                      onChange={(e) =>
                        setSetting({
                          [g.maxSizeSetting ?? ""]: e.toString(),
                        })
                      }
                    />
                    <NoMarginHelperText>{t("settings.thumbMaxSizeDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              )}
              {g.inputs?.map((input) => (
                <SettingForm key={input.name} lgWidth={12} title={t(`settings.${input.label}`)}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      value={values[input.name]}
                      onChange={(e) =>
                        setSetting({
                          [input.name]: e.target.value,
                        })
                      }
                      required={!!input.required}
                    />
                    <NoMarginHelperText>{t(`settings.${input.des}`)}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              ))}
            </SettingSectionContent>
          </AccordionDetails>
        </StyledAccordion>
      ))}
    </Box>
  );
};

export default Generators;
