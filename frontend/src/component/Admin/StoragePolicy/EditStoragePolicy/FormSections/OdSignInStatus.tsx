import { Box, Typography } from "@mui/material";
import { useContext, useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { getPolicyOauthCredentialRefreshTime, getPolicyOauthUrl } from "../../../../../api/api";
import { OauthCredentialStatus } from "../../../../../api/dashboard";
import { useAppDispatch } from "../../../../../redux/hooks";
import FacebookCircularProgress from "../../../../Common/CircularProgress";
import { SecondaryButton } from "../../../../Common/StyledComponents";
import TimeBadge from "../../../../Common/TimeBadge";
import CheckCircleFilled from "../../../../Icons/CheckCircleFilled";
import DismissCircleFilled from "../../../../Icons/DismissCircleFilled";
import OneDriveAuthDialog from "../../Wizards/OneDrive/OneDriveAuthDialog";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";

const OdSignInStatus = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);
  const dispatch = useAppDispatch();
  const [status, setStatus] = useState<OauthCredentialStatus | undefined>(undefined);
  const [loading, setLoading] = useState<boolean>(false);
  const [authorizing, setAuthorizing] = useState<boolean>(false);
  const [authDialogOpen, setAuthDialogOpen] = useState<boolean>(false);

  useEffect(() => {
    if (!values.access_key) {
      setStatus({ valid: false, last_refresh_time: "" });
      return;
    }
    setLoading(true);
    dispatch(getPolicyOauthCredentialRefreshTime(values.id.toString()))
      .then((res) => {
        setStatus(res);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [values.id, values.secret_key]);

  const authorized = !loading && status && !!status.last_refresh_time;

  const handleOpenAuthDialog = () => {
    setAuthDialogOpen(true);
  };

  const handleCloseAuthDialog = () => {
    setAuthDialogOpen(false);
  };

  const handleConfirmAuth = (appId: string, appSecret: string) => {
    setAuthorizing(true);
    dispatch(getPolicyOauthUrl({ id: values.id, secret: appSecret, app_id: appId }))
      .then((res) => {
        window.location.href = res;
      })
      .catch(() => {
        setAuthorizing(false);
      });
  };

  return (
    <Box sx={{ mt: 1 }}>
      {loading && <FacebookCircularProgress size={30} />}
      {!loading && (
        <>
          {!authorized && (
            <Typography variant="body2" color="warning" sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <DismissCircleFilled />
              {t("policy.notGranted")}
            </Typography>
          )}
          {authorized && (
            <Typography variant="body2" color="success" sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <CheckCircleFilled />
              <Trans
                i18nKey="policy.granted"
                ns="dashboard"
                components={[<TimeBadge datetime={status.last_refresh_time} variant="inherit" />]}
              />
            </Typography>
          )}
          <SecondaryButton sx={{ mt: 1 }} variant="contained" onClick={handleOpenAuthDialog} loading={authorizing}>
            {t(authorized ? "policy.authorizeAgain" : "policy.authorizeNow")}
          </SecondaryButton>
        </>
      )}

      <OneDriveAuthDialog
        open={authDialogOpen}
        onClose={handleCloseAuthDialog}
        onConfirm={handleConfirmAuth}
        initialAppId={values.bucket_name}
        initialAppSecret={values.secret_key}
      />
    </Box>
  );
};

export default OdSignInStatus;
