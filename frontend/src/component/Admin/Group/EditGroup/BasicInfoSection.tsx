import { Alert, FormControl, FormControlLabel, Switch, Typography } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { GroupEnt, StoragePolicy } from "../../../../api/dashboard";
import { GroupPermission } from "../../../../api/user";
import Boolset from "../../../../util/boolset";
import SizeInput from "../../../Common/SizeInput";
import { DenseFilledTextField } from "../../../Common/StyledComponents";
import InPrivate from "../../../Icons/InPrivate";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { AnonymousGroupID } from "../GroupRow";
import { GroupSettingContext } from "./GroupSettingWrapper";
import PolicyMultiSelectionInput from "./PolicyMultiSelectionInput";
import PolicySelectionInput from "./PolicySelectionInput";
const BasicInfoSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setGroup } = useContext(GroupSettingContext);

  const permission = useMemo(() => {
    return new Boolset(values.permissions ?? "");
  }, [values.permissions]);

  const onNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({ ...p, name: e.target.value }));
    },
    [setGroup],
  );

  const onPolicyChange = useCallback(
    (value: number) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        edges: { ...p.edges, storage_policies: { id: value } as StoragePolicy },
      }));
    },
    [setGroup],
  );

  const onAllowedPoliciesChange = useCallback(
    (value: number[]) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        edges: { ...p.edges, allowed_policies: value.map((id) => ({ id }) as StoragePolicy) },
      }));
    },
    [setGroup],
  );

  const onMaxStorageChange = useCallback(
    (size: number) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        max_storage: size ? size : undefined,
      }));
    },
    [setGroup],
  );

  const onIsAdminChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.is_admin, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onSectionChange = useCallback(
    (bit: number) => (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(bit, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onWhitelistChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const list = e.target.value
        .split("\n")
        .map((l) => l.trim())
        .filter((l) => l != "");
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: { ...p.settings, login_ip_whitelist: list.length > 0 ? list : undefined },
      }));
    },
    [setGroup],
  );

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("policy.basicInfo")}
      </Typography>
      <SettingSectionContent>
        {values?.id == AnonymousGroupID && (
          <SettingForm lgWidth={5}>
            <Alert icon={<InPrivate fontSize="inherit" />} severity="info">
              {t("group.anonymousHint")}
            </Alert>
          </SettingForm>
        )}
        <SettingForm title={t("group.nameOfGroup")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField required value={values.name} onChange={onNameChange} />
            <NoMarginHelperText>{t("group.nameOfGroupDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {values?.id != AnonymousGroupID && (
          <>
            <SettingForm title={t("group.availablePolicies")} lgWidth={5}>
              <PolicySelectionInput value={values.edges.storage_policies?.id ?? 0} onChange={onPolicyChange} />
              <NoMarginHelperText>{t("group.availablePoliciesDes")}</NoMarginHelperText>
              <NoMarginHelperText> {t("group.availablePolicyDesPro")}
              </NoMarginHelperText>
            </SettingForm>
            <SettingForm title={t("group.switchablePolicies")} lgWidth={5}>
              <PolicyMultiSelectionInput
                value={(values.edges.allowed_policies ?? []).map((p) => p.id)}
                onChange={onAllowedPoliciesChange}
              />
              <NoMarginHelperText>{t("group.switchablePoliciesDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={!!values.settings?.weighted_policies}
                      onChange={(e) =>
                        setGroup((p: GroupEnt) => ({
                          ...p,
                          settings: { ...p.settings, weighted_policies: e.target.checked ? true : undefined },
                        }))
                      }
                    />
                  }
                  label={t("group.weightedPolicies")}
                />
                <NoMarginHelperText>{t("group.weightedPoliciesDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("group.initialStorageQuota")} lgWidth={5}>
              <FormControl fullWidth>
                <SizeInput
                  variant={"outlined"}
                  required
                  value={values.max_storage ?? 0}
                  onChange={onMaxStorageChange}
                />
                <NoMarginHelperText>{t("group.initialStorageQuotaDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  disabled={values.id == 1}
                  control={<Switch checked={permission.enabled(GroupPermission.is_admin)} onChange={onIsAdminChange} />}
                  label={t("group.isAdmin")}
                />
                <NoMarginHelperText>{t("group.isAdminDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            {!permission.enabled(GroupPermission.is_admin) && (
              <SettingForm title={t("group.delegatedAdmin")} lgWidth={5}>
                <FormControl fullWidth>
                  <NoMarginHelperText>{t("group.delegatedAdminDes")}</NoMarginHelperText>
                  {(
                    [
                      [GroupPermission.admin_users, "group.sectionUsers"],
                      [GroupPermission.admin_groups, "group.sectionGroups"],
                      [GroupPermission.admin_files, "group.sectionFiles"],
                      [GroupPermission.admin_shares, "group.sectionShares"],
                      [GroupPermission.admin_storage, "group.sectionStorage"],
                      [GroupPermission.admin_queue, "group.sectionQueue"],
                      [GroupPermission.admin_settings, "group.sectionSettings"],
                      [GroupPermission.admin_payment, "group.sectionPayment"],
                      [GroupPermission.admin_events, "group.sectionEvents"],
                      [GroupPermission.admin_reports, "group.sectionReports"],
                    ] as [number, string][]
                  ).map(([bit, label]) => (
                    <FormControlLabel
                      key={bit}
                      control={<Switch checked={permission.enabled(bit)} onChange={onSectionChange(bit)} />}
                      label={t(label)}
                    />
                  ))}
                </FormControl>
              </SettingForm>
            )}
            <SettingForm title={t("group.loginIPWhitelist")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  multiline
                  minRows={2}
                  value={(values?.settings?.login_ip_whitelist ?? []).join("\n")}
                  onChange={onWhitelistChange}
                  placeholder={"10.0.0.0/8"}
                />
                <NoMarginHelperText>{t("group.loginIPWhitelistDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default BasicInfoSection;
