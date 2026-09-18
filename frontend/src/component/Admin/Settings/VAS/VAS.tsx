import {
  Box,
  Button,
  FormControl,
  FormControlLabel,
  InputAdornment,
  Link,
  Stack,
  Switch,
  Typography,
} from "@mui/material";
import { bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useContext, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import SettingForm, { ProChip } from "../../../Pages/Setting/SettingForm.tsx";
import ProDialog from "../../Common/ProDialog.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import GiftCodes from "./GiftCodes.tsx";
import GroupProducts from "./GroupProducts.tsx";
import PaymentProviders from "./PaymentProviders.tsx";
import StorageProducts from "./StorageProducts.tsx";
interface CurrencyOption {
  code: string;
  symbol: string;
  unit: number;
  label: string;
}

const VAS = () => {
  const { t } = useTranslation("dashboard");
  const [proOpen, setProOpen] = useState(false);
  const { formRef, setSettings, values } = useContext(SettingContext);
  const currencyPopupState = usePopupState({
    variant: "popover",
    popupId: "currencySelector",
  });
  const paymentConfig = useMemo(() => JSON.parse(values.payment || "{}"), [values.payment]);
  const storageProducts = useMemo(() => values.storage_products || "[]", [values.storage_products]);
  const groupSellData = useMemo(() => values.group_sell_data || "[]", [values.group_sell_data]);

  const onProClick = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    setProOpen(true);
  };

  return (
    <Box component={"form"} ref={formRef}>
      <ProDialog open={proOpen} onClose={() => setProOpen(false)} />
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.creditAndVAS")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <SettingSectionContent onClick={onProClick}>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel control={<Switch checked={false} />} label={t("settings.enableCredit")} />
                <NoMarginHelperText>{t("settings.enableCreditDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <Stack spacing={2}>
              <SettingForm title={t("settings.creditPrice")} lgWidth={5}>
                <FormControl fullWidth>
                  <DenseFilledTextField type="number" slotProps={{ input: { readOnly: true } }} value={1} />
                  <NoMarginHelperText>{t("settings.creditPriceDes")}</NoMarginHelperText>
                </FormControl>
              </SettingForm>

              <SettingForm title={t("settings.shareScoreRate")} lgWidth={5}>
                <FormControl fullWidth>
                  <DenseFilledTextField type="number" value={80} slotProps={{ input: { readOnly: true } }} />
                  <NoMarginHelperText>{t("settings.shareScoreRateDes")}</NoMarginHelperText>
                </FormControl>
              </SettingForm>
            </Stack>

            <SettingForm title={t("vas.banBufferPeriod")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField type="number" slotProps={{ input: { readOnly: true } }} value={864000} />
                <NoMarginHelperText>{t("vas.banBufferPeriodDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.cronNotifyUser")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField value={"@every 1h"} slotProps={{ input: { readOnly: true } }} />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.cronDes"
                    values={{
                      des: t("settings.cronNotifyUserDes"),
                    }}
                    ns={"dashboard"}
                    components={[<Link href="https://crontab.guru/" target="_blank" rel="noopener noreferrer" />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.cronBanUser")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField value={"@every 1h"} slotProps={{ input: { readOnly: true } }} />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.cronDes"
                    values={{
                      des: t("settings.cronBanUserDes"),
                    }}
                    ns={"dashboard"}
                    components={[<Link href="https://crontab.guru/" target="_blank" rel="noopener noreferrer" />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel control={<Switch checked={false} />} label={t("settings.anonymousPurchase")} />
                <NoMarginHelperText>{t("settings.anonymousPurchaseDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel control={<Switch checked={false} />} label={t("settings.shopNavEnabled")} />
                <NoMarginHelperText>{t("settings.shopNavEnabledDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>

        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.paymentSettings")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <SettingSectionContent onClick={onProClick}>
            <SettingForm title={t("settings.currencyCode")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value="USD"
                  slotProps={{
                    input: {
                      readOnly: true,
                      endAdornment: (
                        <InputAdornment position="end">
                          <Button {...bindTrigger(currencyPopupState)}>{t("settings.selectCurrency")}</Button>
                        </InputAdornment>
                      ),
                    },
                  }}
                />
                <NoMarginHelperText>{t("settings.currencyCodeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.currencySymbol")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField value={"$"} slotProps={{ input: { readOnly: true } }} />
                <NoMarginHelperText>{t("settings.currencySymbolDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.currencyUnit")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField type="number" value={100} slotProps={{ input: { readOnly: true } }} />
                <NoMarginHelperText>{t("settings.currencyUnitDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <Box sx={{ mt: 2 }}>
              <Typography variant="subtitle1" gutterBottom>
                {t("settings.paymentProviders")}
              </Typography>
              <SettingForm lgWidth={6}>
                <PaymentProviders />
              </SettingForm>
            </Box>
          </SettingSectionContent>
        </SettingSection>

        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.storageProductSettings")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <SettingSectionContent onClick={onProClick}>
            <SettingForm lgWidth={12}>
              <FormControl fullWidth>
                <StorageProducts />
                <NoMarginHelperText>{t("settings.storageProductsDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>

        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.groupProductSettings")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <SettingSectionContent onClick={onProClick}>
            <SettingForm lgWidth={12}>
              <FormControl fullWidth>
                <GroupProducts />
                <NoMarginHelperText>{t("settings.groupProductsDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>

        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("giftCodes.giftCodesSettings")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <SettingSectionContent onClick={onProClick}>
            <GiftCodes storageProductsConfig={storageProducts} groupProductsConfig={groupSellData} />
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default VAS;
