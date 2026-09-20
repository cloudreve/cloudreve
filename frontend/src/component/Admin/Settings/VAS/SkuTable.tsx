import {
  Box,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Switch,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { adminDeleteSku, adminListSkus, adminUpsertSku, getGroupList } from "../../../../api/api.ts";
import { GroupEnt, Sku } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { sizeToString } from "../../../../util/index.ts";
import {
  DenseFilledTextField,
  NoWrapCell,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Edit from "../../../Icons/Edit.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import LocalizedFields from "../LocalizedFields.tsx";
import { NoMarginHelperText } from "../Settings.tsx";

const DAY_SECONDS = 86400;

export interface SkuTableProps {
  type: "storage" | "group" | "traffic";
}

interface SkuForm {
  id: number;
  name: string;
  amount: number;
  durationDays: number;
  price: number;
  allowPoints: boolean;
  points: number;
  label: string;
  des: string;
  nameI18n: { [lang: string]: string };
  desI18n: { [lang: string]: string };
  enabled: boolean;
}

const emptyForm = (type: string): SkuForm => ({
  id: 0,
  name: "",
  amount: 0,
  durationDays: type === "group" ? 30 : type === "storage" ? 365 : 0,
  price: 0,
  allowPoints: false,
  points: 0,
  label: "",
  des: "",
  nameI18n: {},
  desI18n: {},
  enabled: true,
});

const nonEmpty = (m: { [lang: string]: string }) => (Object.keys(m).length > 0 ? m : undefined);

const SkuTable = ({ type }: SkuTableProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [skus, setSkus] = useState<Sku[] | undefined>(undefined);
  const [groups, setGroups] = useState<GroupEnt[]>([]);
  const [form, setForm] = useState<SkuForm | undefined>(undefined);
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    dispatch(adminListSkus()).then((res) => setSkus(res.filter((s) => s.type === type)));
  }, [type]);

  useEffect(() => {
    load();
    dispatch(getGroupList({ page: 1, page_size: 200, order_by: "id", order_direction: "asc" })).then((res) =>
      setGroups(res.groups),
    );
  }, [load]);

  const groupName = useCallback(
    (id: number) => groups.find((g) => g.id === id)?.name ?? `#${id}`,
    [groups],
  );

  const openEdit = (s?: Sku) => {
    if (!s) {
      setForm(emptyForm(type));
      return;
    }
    setForm({
      id: s.id,
      name: s.name,
      amount: s.amount,
      durationDays: Math.round((s.duration ?? 0) / DAY_SECONDS),
      price: s.price ?? 0,
      allowPoints: s.points != null,
      points: s.points ?? 0,
      label: s.label ?? "",
      des: s.des ?? "",
      nameI18n: s.name_i18n ?? {},
      desI18n: s.des_i18n ?? {},
      enabled: s.enabled,
    });
  };

  const onSave = () => {
    if (!form) {
      return;
    }
    setSaving(true);
    dispatch(
      adminUpsertSku({
        id: form.id,
        name: form.name,
        type,
        amount: form.amount,
        // Traffic packs are permanent; duration is not applicable.
        duration: type === "traffic" ? 0 : Math.max(0, form.durationDays) * DAY_SECONDS,
        price: Math.max(0, form.price),
        points: form.allowPoints ? Math.max(1, form.points) : undefined,
        label: form.label || undefined,
        des: form.des || undefined,
        name_i18n: nonEmpty(form.nameI18n),
        des_i18n: nonEmpty(form.desI18n),
        enabled: form.enabled,
        weight: 0,
      }),
    )
      .then(() => {
        setForm(undefined);
        load();
      })
      .finally(() => setSaving(false));
  };

  const onDelete = (id: number) => {
    dispatch(adminDeleteSku(id)).then(load);
  };

  const amountLabel = (s: Sku) => (type === "group" ? groupName(s.amount) : sizeToString(s.amount));

  return (
    <Stack spacing={2}>
      <Box>
        <SecondaryButton variant="contained" startIcon={<Add />} onClick={() => openEdit()}>
          {type === "group"
            ? t("vas.addMembership")
            : type === "traffic"
              ? t("vas.addTrafficPack")
              : t("vas.addStoragePack")}
        </SecondaryButton>
      </Box>

      <StyledTableContainerPaper>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <NoWrapCell>{t("vas.name")}</NoWrapCell>
                <NoWrapCell>{type === "group" ? t("vas.group") : t("vas.size")}</NoWrapCell>
                <NoWrapCell>{t("vas.duration")}</NoWrapCell>
                <NoWrapCell>{t("vas.price")}</NoWrapCell>
                <NoWrapCell>{t("vas.priceCredits")}</NoWrapCell>
                <NoWrapCell>{t("vas.status")}</NoWrapCell>
                <NoWrapCell align="right">{t("vas.actions")}</NoWrapCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {skus?.map((s) => (
                <TableRow key={s.id}>
                  <NoWrapCell>
                    {s.name}
                    {s.label && <Chip size="small" color="primary" label={s.label} sx={{ ml: 1 }} />}
                  </NoWrapCell>
                  <NoWrapCell>{amountLabel(s)}</NoWrapCell>
                  <NoWrapCell>
                    {(s.duration ?? 0) > 0
                      ? t("application:vas.validDurationDays", { num: Math.round((s.duration ?? 0) / DAY_SECONDS) })
                      : t("application:shop.permanent")}
                  </NoWrapCell>
                  <NoWrapCell>{s.price ?? 0}</NoWrapCell>
                  <NoWrapCell>{s.points != null ? s.points : "-"}</NoWrapCell>
                  <NoWrapCell>
                    <Chip
                      size="small"
                      color={s.enabled ? "success" : "default"}
                      label={s.enabled ? t("vas.enable") : t("vas.no")}
                    />
                  </NoWrapCell>
                  <NoWrapCell align="right">
                    <IconButton size="small" onClick={() => openEdit(s)}>
                      <Edit fontSize="small" />
                    </IconButton>
                    <IconButton size="small" onClick={() => onDelete(s.id)}>
                      <Dismiss fontSize="small" />
                    </IconButton>
                  </NoWrapCell>
                </TableRow>
              ))}
              {(!skus || skus.length === 0) && (
                <TableRow>
                  <NoWrapCell colSpan={7} align="center">
                    <Typography variant="caption" color="text.secondary">
                      {t("application:setting.listEmpty")}
                    </Typography>
                  </NoWrapCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </StyledTableContainerPaper>

      <Dialog open={!!form} onClose={() => setForm(undefined)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {type === "group"
            ? t("vas.editMembership")
            : type === "traffic"
              ? t("vas.editTrafficPack")
              : t("vas.editStoragePack")}
        </DialogTitle>
        <DialogContent>
          {form && (
            <Stack spacing={2} sx={{ mt: 1 }}>
              <SettingForm title={t("vas.productName")}>
                <DenseFilledTextField
                  fullWidth
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
                <NoMarginHelperText>{t("vas.productNameDes")}</NoMarginHelperText>
                <LocalizedFields
                  value={JSON.stringify(form.nameI18n)}
                  onChange={(v) => setForm({ ...form, nameI18n: JSON.parse(v) })}
                />
              </SettingForm>

              {type === "group" ? (
                <SettingForm title={t("vas.purchasableGroups")}>
                  <FormControl fullWidth size="small">
                    <InputLabel>{t("vas.purchasableGroups")}</InputLabel>
                    <Select
                      value={form.amount}
                      label={t("vas.purchasableGroups")}
                      onChange={(e) => setForm({ ...form, amount: e.target.value as number })}
                    >
                      {groups.map((g) => (
                        <MenuItem key={g.id} value={g.id}>
                          {g.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <NoMarginHelperText>{t("vas.groupDes")}</NoMarginHelperText>
                </SettingForm>
              ) : (
                <SettingForm title={type === "traffic" ? t("vas.trafficSize") : t("vas.size")}>
                  <DenseFilledTextField
                    fullWidth
                    type="number"
                    value={form.amount}
                    onChange={(e) => setForm({ ...form, amount: parseInt(e.target.value) || 0 })}
                  />
                  <NoMarginHelperText>
                    {type === "traffic" ? t("vas.trafficSizeDes") : t("vas.packSizeDes")}
                  </NoMarginHelperText>
                </SettingForm>
              )}

              {type !== "traffic" && (
              <SettingForm title={t("vas.durationDay")}>
                <DenseFilledTextField
                  fullWidth
                  type="number"
                  value={form.durationDays}
                  onChange={(e) => setForm({ ...form, durationDays: parseInt(e.target.value) || 0 })}
                />
                <NoMarginHelperText>
                  {type === "storage" ? t("vas.durationDayDes") : t("vas.durationGroupDes")}
                </NoMarginHelperText>
              </SettingForm>
              )}

              <SettingForm title={t("vas.priceYuan")}>
                <DenseFilledTextField
                  fullWidth
                  type="number"
                  value={form.price}
                  onChange={(e) => setForm({ ...form, price: parseInt(e.target.value) || 0 })}
                />
                <NoMarginHelperText>{t("vas.packPriceDes")}</NoMarginHelperText>
              </SettingForm>

              <SettingForm lgWidth={5}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={form.allowPoints}
                      onChange={(e) => setForm({ ...form, allowPoints: e.target.checked })}
                    />
                  }
                  label={t("vas.priceCredits")}
                />
                <NoMarginHelperText>{t("vas.priceCreditsDes")}</NoMarginHelperText>
              </SettingForm>

              {form.allowPoints && (
                <SettingForm title={t("vas.priceCredits")}>
                  <DenseFilledTextField
                    fullWidth
                    type="number"
                    value={form.points}
                    onChange={(e) => setForm({ ...form, points: parseInt(e.target.value) || 0 })}
                  />
                </SettingForm>
              )}

              <SettingForm title={t("vas.highlight")}>
                <DenseFilledTextField
                  fullWidth
                  value={form.label}
                  onChange={(e) => setForm({ ...form, label: e.target.value })}
                />
                <NoMarginHelperText>{t("vas.highlightDes")}</NoMarginHelperText>
              </SettingForm>

              <SettingForm title={t("vas.productDescription")}>
                <DenseFilledTextField
                  fullWidth
                  multiline
                  minRows={3}
                  value={form.des}
                  onChange={(e) => setForm({ ...form, des: e.target.value })}
                />
                <NoMarginHelperText>{t("vas.productDescriptionDes")}</NoMarginHelperText>
                <LocalizedFields
                  multiline
                  rows={2}
                  value={JSON.stringify(form.desI18n)}
                  onChange={(v) => setForm({ ...form, desI18n: JSON.parse(v) })}
                />
              </SettingForm>

              <SettingForm lgWidth={5}>
                <FormControlLabel
                  control={
                    <Switch checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
                  }
                  label={t("vas.enable")}
                />
              </SettingForm>
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          <SecondaryButton onClick={() => setForm(undefined)}>{t("common:cancel")}</SecondaryButton>
          <SecondaryButton variant="contained" onClick={onSave} disabled={saving || !form?.name || !form?.amount}>
            {t("common:ok")}
          </SecondaryButton>
        </DialogActions>
      </Dialog>
    </Stack>
  );
};

export default SkuTable;
