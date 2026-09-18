import { Alert, Box, Collapse, FormControlLabel, Link, ListItemText, Stack, Switch, Typography } from "@mui/material";
import FormControl from "@mui/material/FormControl";
import { useCallback, useContext, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import Extractors from "./Extractors.tsx";
import Generators from "./Generators.tsx";
import MagicVarDialog from "../../Common/MagicVarDialog";
import { thumbnailMagicVars } from "../../StoragePolicy/EditStoragePolicy/FormSections/magicVars.ts";

const Media = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);
  const [magicVarDialogOpen, setMagicVarDialogOpen] = useState(false);

  const handleThumbMagicVarClick = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setMagicVarDialogOpen(true);
  }, []);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.thumbnails")}
          </Typography>
          <Typography variant="subtitle1" gutterBottom>
            {t("settings.thumbnailBasic")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.thumbWidth")} lgWidth={5}>
              <FormControl>
                <DenseFilledTextField
                  type="number"
                  required
                  inputProps={{ min: 1, step: 1 }}
                  value={values.thumb_width}
                  onChange={(e) => {
                    setSettings({
                      thumb_width: e.target.value,
                    });
                  }}
                />
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.thumbHeight")} lgWidth={5}>
              <FormControl>
                <DenseFilledTextField
                  type="number"
                  required
                  inputProps={{ min: 1, step: 1 }}
                  value={values.thumb_height}
                  onChange={(e) => {
                    setSettings({
                      thumb_height: e.target.value,
                    });
                  }}
                />
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.thumbPath")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.thumb_entity_suffix}
                  onChange={(e) => {
                    setSettings({
                      thumb_entity_suffix: e.target.value,
                    });
                  }}
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.notAppliedToNativeGenerator"
                    ns="dashboard"
                    values={{ prefix: t("settings.thumbPathDes") }}
                    components={[<Link href="#" onClick={handleThumbMagicVarClick} />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.thumbFormat")} lgWidth={5}>
              <FormControl>
                <DenseSelect
                  value={values.thumb_encode_method}
                  onChange={(e) => {
                    setSettings({
                      thumb_encode_method: e.target.value as string,
                    });
                  }}
                  required
                >
                  {["jpg", "png", "webp"].map((f) => (
                    <SquareMenuItem value={f} key={f}>
                      <ListItemText
                        slotProps={{
                          primary: { variant: "body2" },
                        }}
                      >
                        {f}
                      </ListItemText>
                    </SquareMenuItem>
                  ))}
                </DenseSelect>
                <NoMarginHelperText>
                  {t("settings.notAppliedToOneDriveNativeGenerator", { prefix: t("settings.thumbFormatDes") })}
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <Collapse in={values.thumb_encode_method == "jpg" || values.thumb_encode_method == "webp"} unmountOnExit>
              <SettingForm title={t("settings.thumbQuality")} lgWidth={5}>
                <FormControl>
                  <DenseFilledTextField
                    type="number"
                    required
                    inputProps={{ min: 50, max: 100, step: 1 }}
                    value={values.thumb_encode_quality}
                    onChange={(e) => {
                      setSettings({
                        thumb_encode_quality: e.target.value,
                      });
                    }}
                  />
                  <NoMarginHelperText>
                    {t("settings.notAppliedToOneDriveNativeGenerator", {
                      prefix: t("settings.thumbQualityDes"),
                    })}
                  </NoMarginHelperText>
                </FormControl>
              </SettingForm>
            </Collapse>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.thumb_gc_after_gen)}
                      onChange={(e) =>
                        setSettings({
                          thumb_gc_after_gen: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.thumbGC")}
                />
                <NoMarginHelperText>{t("settings.notAppliedToNativeGenerator", { prefix: "" })}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
          <MagicVarDialog
            open={magicVarDialogOpen}
            onClose={() => setMagicVarDialogOpen(false)}
            vars={thumbnailMagicVars}
          />
          <Typography variant="subtitle1" gutterBottom sx={{ mt: 1 }}>
            {t("settings.generators")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={6}>
              <Alert severity="info" sx={{ mb: 1 }}>
                <Trans
                  ns="dashboard"
                  i18nKey="settings.generatorProxyWarning"
                  components={[<Link href="https://docs.cloudreve.org/usage/thumbnails" target="_blank" />]}
                />
              </Alert>
              <Generators values={values} setSetting={setSettings} />
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.extractMediaMeta")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={6}>
              <Alert severity="info" sx={{ mb: 1 }}>
                <Trans
                  ns="dashboard"
                  i18nKey="settings.extractMediaMetaDes"
                  components={[<Link href="https://docs.cloudreve.org/usage/media-meta" target="_blank" />]}
                />
              </Alert>
              <Extractors values={values} setSetting={setSettings} />
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default Media;
