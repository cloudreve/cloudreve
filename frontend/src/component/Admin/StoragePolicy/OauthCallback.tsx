import { Box, Container } from "@mui/material";
import { useSnackbar } from "notistack";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { finishOauthCallback } from "../../../api/api";
import { useAppDispatch } from "../../../redux/hooks";
import { useQuery } from "../../../util";
import FacebookCircularProgress from "../../Common/CircularProgress";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
const OauthCallback = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();

  const query = useQuery();

  useEffect(() => {
    const code = query.get("code");
    const state = query.get("state");
    if (code && state) {
      dispatch(finishOauthCallback({ code, state })).finally(() => {
        navigate(`/admin/policy/${state}`);
      });
    } else {
      enqueueSnackbar(t("policy.oauthCallbackFailed"), {
        variant: "error",
        action: DefaultCloseAction,
      });
    }
  }, []);

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.storagePolicy")} />
      </Container>

      <Box
        sx={{
          pt: 15,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
        }}
      >
        <FacebookCircularProgress />
      </Box>
    </PageContainer>
  );
};

export default OauthCallback;
