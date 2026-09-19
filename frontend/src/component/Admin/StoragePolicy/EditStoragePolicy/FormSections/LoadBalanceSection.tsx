import {
  Box,
  FormControl,
  IconButton,
  SelectChangeEvent,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getStoragePolicyList } from "../../../../../api/api";
import { LBPolicyRef, StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DenseFilledTextField, DenseSelect, SecondaryButton } from "../../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../../FileManager/ContextMenu/ContextMenu";
import Dismiss from "../../../../Icons/Dismiss";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../../Settings/Settings";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";

// LoadBalanceSection edits the weighted child policies of a load_balance
// policy. Only concrete (non-load-balance) policies can be children.
const LoadBalanceSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);
  const dispatch = useAppDispatch();
  const [policies, setPolicies] = useState<StoragePolicy[]>([]);

  const isLB = values.type === PolicyType.load_balance;
  const refs = values.settings?.lb_policies ?? [];

  useEffect(() => {
    if (!isLB) {
      return;
    }
    dispatch(getStoragePolicyList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" })).then(
      (res) => {
        setPolicies(res.policies.filter((p) => p.type !== PolicyType.load_balance && p.id !== values.id));
      },
    );
  }, [isLB]);

  const setRefs = useCallback(
    (next: LBPolicyRef[]) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, lb_policies: next },
      }));
    },
    [setPolicy],
  );

  const addChild = useCallback(() => {
    const used = new Set(refs.map((r) => r.policy));
    const candidate = policies.find((p) => !used.has(p.id));
    if (!candidate) {
      return;
    }
    setRefs([...refs, { policy: candidate.id, weight: 1 }]);
  }, [refs, policies, setRefs]);

  const onChildChange = useCallback(
    (index: number, e: SelectChangeEvent<unknown>) => {
      const next = refs.slice();
      next[index] = { ...next[index], policy: e.target.value as number };
      setRefs(next);
    },
    [refs, setRefs],
  );

  const onWeightChange = useCallback(
    (index: number, v: string) => {
      const w = parseInt(v);
      const next = refs.slice();
      next[index] = { ...next[index], weight: isNaN(w) ? undefined : Math.max(1, w) };
      setRefs(next);
    },
    [refs, setRefs],
  );

  const removeChild = useCallback(
    (index: number) => {
      setRefs(refs.filter((_, i) => i !== index));
    },
    [refs, setRefs],
  );

  if (!isLB) {
    return null;
  }

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("policy.loadBalance")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("policy.lbChildren")} lgWidth={12}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>{t("policy.lbChildPolicy")}</TableCell>
                <TableCell width={140}>{t("policy.lbWeight")}</TableCell>
                <TableCell width={48} />
              </TableRow>
            </TableHead>
            <TableBody>
              {refs.map((r, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <FormControl fullWidth size="small">
                      <DenseSelect value={r.policy} onChange={(e) => onChildChange(i, e)}>
                        {policies.map((p) => (
                          <SquareMenuItem
                            key={p.id}
                            value={p.id}
                            disabled={refs.some((x, xi) => xi !== i && x.policy === p.id)}
                          >
                            {p.name}
                          </SquareMenuItem>
                        ))}
                      </DenseSelect>
                    </FormControl>
                  </TableCell>
                  <TableCell>
                    <DenseFilledTextField
                      type="number"
                      size="small"
                      inputProps={{ min: 1 }}
                      value={r.weight ?? 1}
                      onChange={(e) => onWeightChange(i, e.target.value)}
                    />
                  </TableCell>
                  <TableCell>
                    <IconButton size="small" onClick={() => removeChild(i)}>
                      <Dismiss fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Box sx={{ mt: 1 }}>
            <SecondaryButton onClick={addChild} disabled={refs.length >= policies.length}>
              {t("policy.lbAddChild")}
            </SecondaryButton>
          </Box>
          <NoMarginHelperText>{t("policy.lbChildrenDes")}</NoMarginHelperText>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default LoadBalanceSection;
