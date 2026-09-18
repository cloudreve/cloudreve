import { ExpandMoreRounded } from "@mui/icons-material";
import { LoadingButton } from "@mui/lab";
import {
  AccordionDetails,
  Box,
  FormControl,
  FormControlLabel,
  InputAdornment,
  Switch,
  Typography,
} from "@mui/material";
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

export interface ExtractorsProps {
  values: {
    [key: string]: any;
  };
  setSetting: (v: { [key: string]: any }) => void;
}

interface ExtractorRenderProps {
  name: string;
  des: string;
  enableFlag?: string;
  executableSetting?: string;
  maxSizeLocalSetting?: string;
  maxSizeRemoteSetting?: string;
  additionalSettings?: {
    name: string;
    label: string;
    des: string;
    type?: "switch";
  }[];
}

const extractors: ExtractorRenderProps[] = [
  {
    name: "exif",
    des: "exifDes",
    enableFlag: "media_meta_exif",
    maxSizeLocalSetting: "media_meta_exif_size_local",
    maxSizeRemoteSetting: "media_meta_exif_size_remote",
    additionalSettings: [
      {
        name: "media_meta_exif_brute_force",
        label: "exifBruteForce",
        des: "exifBruteForceDes",
        type: "switch",
      },
    ],
  },
  {
    name: "music",
    des: "musicDes",
    enableFlag: "media_meta_music",
    maxSizeLocalSetting: "media_meta_music_size_local",
    maxSizeRemoteSetting: "media_exif_music_size_remote",
  },
  {
    name: "ffprobe",
    des: "ffprobeDes",
    enableFlag: "media_meta_ffprobe",
    executableSetting: "media_meta_ffprobe_path",
    maxSizeLocalSetting: "media_meta_ffprobe_size_local",
    maxSizeRemoteSetting: "media_meta_ffprobe_size_remote",
  },
  {
    name: "geocoding",
    des: "geocodingDes",
    enableFlag: "media_meta_geocoding",
    additionalSettings: [
      {
        name: "media_meta_geocoding_mapbox_ak",
        label: "mapboxAK",
        des: "mapboxAKDes",
      },
    ],
  },
];

const Extractors = ({ values, setSetting }: ExtractorsProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [testing, setTesting] = useState(false);
  const { enqueueSnackbar } = useSnackbar();

  const handleEnableChange = (name: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
    setSetting({
      [name]: e.target.checked ? "1" : "0",
    });

    const newValues = { ...values, [name]: e.target.checked ? "1" : "0" };
    if (isTrueVal(newValues["media_meta_geocoding"]) && !isTrueVal(newValues["media_meta_exif"])) {
      enqueueSnackbar({
        message: t("settings.geocodingDependencyWarning"),
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
      {extractors.map((e) => (
        <StyledAccordion key={e.name} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreRounded />}>
            <FormControlLabel
              onClick={(event) => event.stopPropagation()}
              onFocus={(event) => event.stopPropagation()}
              control={
                <StyledCheckbox
                  size={"small"}
                  checked={isTrueVal(values[e.enableFlag ?? ""])}
                  onChange={handleEnableChange(e.enableFlag ?? "")}
                />
              }
              label={t(`settings.${e.name}`)}
            />
          </AccordionSummary>
          <AccordionDetails sx={{ display: "block" }}>
            <Typography color="textSecondary" variant={"body2"}>
              {t(`settings.${e.des}`)}
            </Typography>
            <SettingSectionContent sx={{ mt: 2 }}>
              {e.executableSetting && (
                <SettingForm lgWidth={12} title={t("settings.executable")}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      required
                      value={values[e.executableSetting]}
                      InputProps={{
                        endAdornment: (
                          <InputAdornment position="end">
                            <LoadingButton
                              onClick={() => doTest(e.name, values[e.executableSetting ?? ""])}
                              loading={testing}
                              color="primary"
                            >
                              <span>{t("settings.executableTest")}</span>
                            </LoadingButton>
                          </InputAdornment>
                        ),
                      }}
                      onChange={(ev) =>
                        setSetting({
                          [e.executableSetting ?? ""]: ev.target.value,
                        })
                      }
                    />
                    <NoMarginHelperText>{t("settings.executableDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              )}
              {e.maxSizeLocalSetting && (
                <SettingForm lgWidth={12} title={t("settings.maxSizeLocal")}>
                  <FormControl fullWidth>
                    <SizeInput
                      variant={"outlined"}
                      required
                      value={parseInt(values[e.maxSizeLocalSetting ?? ""]) ?? 0}
                      onChange={(v) =>
                        setSetting({
                          [e.maxSizeLocalSetting ?? ""]: v.toString(),
                        })
                      }
                    />
                    <NoMarginHelperText>{t("settings.maxSizeLocalDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              )}
              {e.maxSizeRemoteSetting && (
                <SettingForm lgWidth={12} title={t("settings.maxSizeRemote")}>
                  <FormControl fullWidth>
                    <SizeInput
                      variant={"outlined"}
                      required
                      value={parseInt(values[e.maxSizeRemoteSetting ?? ""]) ?? 0}
                      onChange={(v) =>
                        setSetting({
                          [e.maxSizeRemoteSetting ?? ""]: v.toString(),
                        })
                      }
                    />
                    <NoMarginHelperText>{t("settings.maxSizeRemoteDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              )}
              {e.additionalSettings?.map((setting) => (
                <SettingForm key={setting.name} lgWidth={12}>
                  <FormControl fullWidth>
                    {setting.type === "switch" ? (
                      <FormControlLabel
                        control={
                          <Switch
                            checked={isTrueVal(values[setting.name])}
                            onChange={(ev) =>
                              setSetting({
                                [setting.name]: ev.target.checked ? "1" : "0",
                              })
                            }
                          />
                        }
                        label={t(`settings.${setting.label}`)}
                      />
                    ) : (
                      <DenseFilledTextField
                        label={t(`settings.${setting.label}`)}
                        required={isTrueVal(values[e.enableFlag ?? ""])}
                        value={values[setting.name]}
                        onChange={(ev) =>
                          setSetting({
                            [setting.name]: ev.target.value,
                          })
                        }
                      />
                    )}
                    <NoMarginHelperText>{t(`settings.${setting.des}`)}</NoMarginHelperText>
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

export default Extractors;
