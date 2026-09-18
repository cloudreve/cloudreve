import { Alert, FormControl, FormControlLabel, Switch, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useContext, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { testNode } from "../../../../api/api";
import { Node, NodeStatus, NodeType } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { Code } from "../../../Common/Code.tsx";
import { EndpointInput } from "../../Common/EndpointInput";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { NodeSettingContext } from "./NodeSettingWrapper";
const BasicInfoSection = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const { values, setNode } = useContext(NodeSettingContext);
  const [testNodeLoading, setTestNodeLoading] = useState(false);

  const onNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({ ...p, name: e.target.value }));
    },
    [setNode],
  );

  const onServerChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({ ...p, server: e.target.value }));
    },
    [setNode],
  );

  const onSlaveKeyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({ ...p, slave_key: e.target.value }));
    },
    [setNode],
  );

  const onWeightChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const weight = parseInt(e.target.value);
      setNode((p: Node) => ({ ...p, weight: isNaN(weight) ? 1 : weight }));
    },
    [setNode],
  );

  const onStatusChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        status: e.target.checked ? NodeStatus.active : NodeStatus.suspended,
      }));
    },
    [setNode],
  );

  const isActive = useMemo(() => {
    return values.status === NodeStatus.active;
  }, [values.status]);

  const nodeTypeText = useMemo(() => {
    return values.type === NodeType.master ? t("node.master") : t("node.slave");
  }, [values.type, t]);

  const onTestNode = useCallback(() => {
    setTestNodeLoading(true);
    dispatch(testNode({ node: values }))
      .then((res) => {
        enqueueSnackbar(t("node.testNodeSuccess"), { variant: "success", action: DefaultCloseAction });
      })
      .finally(() => {
        setTestNodeLoading(false);
      });
  }, [dispatch, values]);

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("policy.basicInfo")}
      </Typography>
      <SettingSectionContent>
        {values.type === NodeType.master && (
          <SettingForm lgWidth={5}>
            <Alert severity="info">{t("node.thisIsMasterNodes")}</Alert>
          </SettingForm>
        )}
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              disabled={values.type === NodeType.master}
              control={<Switch checked={isActive} onChange={onStatusChange} />}
              label={t("node.enableNode")}
            />
            <NoMarginHelperText>{t("node.enableNodeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("node.name")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField required value={values.name} onChange={onNameChange} />
            <NoMarginHelperText>{t("node.nameNode")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("node.type")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField disabled value={nodeTypeText} />
          </FormControl>
        </SettingForm>
        {values.type === NodeType.slave && (
          <>
            <SettingForm title={t("node.server")} lgWidth={5}>
              <FormControl fullWidth>
                <EndpointInput
                  fullWidth
                  enforceProtocol
                  required
                  value={values.server}
                  onChange={onServerChange}
                  variant={"outlined"}
                />
                <NoMarginHelperText>{t("node.serverDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("node.slaveSecret")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField required value={values.slave_key} onChange={onSlaveKeyChange} />
                <NoMarginHelperText>
                  <Trans i18nKey="node.slaveSecretDes" ns="dashboard" components={[<Code />, <Code />]} />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </>
        )}
        <SettingForm title={t("node.loadBalancerRank")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField
              type="number"
              slotProps={{
                htmlInput: {
                  inputProps: {
                    min: 1,
                  },
                },
              }}
              required
              value={values.weight}
              onChange={onWeightChange}
            />
            <NoMarginHelperText>{t("node.loadBalancerRankDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {values.type === NodeType.slave && (
          <SettingForm lgWidth={5}>
            <SecondaryButton loading={testNodeLoading} variant="contained" onClick={onTestNode}>
              {t("node.testNode")}
            </SecondaryButton>
          </SettingForm>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default BasicInfoSection;
