import { FormControl, Link, Typography } from "@mui/material";
import { useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import SharesInput from "../../Common/SharesInput";
import SettingForm, { ProChip } from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { AnonymousGroupID } from "../GroupRow";
import { GroupSettingContext } from "./GroupSettingWrapper";

const DefaultPinnedSection = () => {
  const { t } = useTranslation("dashboard");
  const { values } = useContext(GroupSettingContext);

  if (values?.id == AnonymousGroupID) {
    return null;
  }

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
        {t("group.defaultPinned")} <ProChip label="Pro" color="primary" size="small" />
      </Typography>
      <SettingSectionContent>
        <SettingForm lgWidth={5} pro>
          <FormControl fullWidth>
            <SharesInput />
            <NoMarginHelperText>
              <Trans
                i18nKey="group.defaultPinnedDes"
                ns={"dashboard"}
                components={[<Link component={RouterLink} to={"/admin/share"} />]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default DefaultPinnedSection;
