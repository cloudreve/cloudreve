import { Delete } from "@mui/icons-material";
import {
  Box,
  Button,
  DialogContent,
  IconButton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  createInvitationCode,
  deleteInvitationCode,
  InvitationCode,
  listInvitationCodes,
} from "../../../api/api";
import { useAppDispatch } from "../../../redux/hooks";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";
import { DenseFilledTextField, NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import Add from "../../Icons/Add";
import ArrowSync from "../../Icons/ArrowSync";
import CopyOutlined from "../../Icons/CopyOutlined";
import SettingForm from "../../Pages/Setting/SettingForm";
import GroupSelectionInput from "../Common/GroupSelectionInput";

export interface InvitationCodeDialogProps {
  open: boolean;
  onClose: () => void;
}

const InvitationCodeDialog = ({ open, onClose }: InvitationCodeDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const [codes, setCodes] = useState<InvitationCode[]>([]);
  const [nextToken, setNextToken] = useState("");
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);

  const [code, setCode] = useState("");
  const [groupId, setGroupId] = useState("");
  const [maxUses, setMaxUses] = useState("1");
  const [expiresAt, setExpiresAt] = useState("");

  const fetchCodes = useCallback(
    (token?: string) => {
      setLoading(true);
      dispatch(listInvitationCodes({ page_size: 20, page_token: token }))
        .then((res) => {
          setCodes((prev) => (token ? [...prev, ...res.codes] : res.codes));
          setNextToken(res.pagination.next_token ?? "");
        })
        .finally(() => setLoading(false));
    },
    [dispatch],
  );

  useEffect(() => {
    if (open) {
      setCodes([]);
      setCode("");
      setGroupId("");
      setMaxUses("1");
      setExpiresAt("");
      fetchCodes();
    }
  }, [open, fetchCodes]);

  const handleCreate = () => {
    setCreating(true);
    dispatch(
      createInvitationCode({
        code: code || undefined,
        group_id: groupId ? parseInt(groupId) : undefined,
        max_uses: maxUses ? parseInt(maxUses) : undefined,
        expires_at: expiresAt ? new Date(expiresAt).toISOString() : undefined,
      }),
    )
      .then(() => {
        setCode("");
        fetchCodes();
        enqueueSnackbar({ message: t("user.invitationCodeCreated"), variant: "success", action: DefaultCloseAction });
      })
      .finally(() => setCreating(false));
  };

  const handleDelete = (id: number) => {
    dispatch(deleteInvitationCode(id)).then(() => fetchCodes());
  };

  const handleCopy = (value: string) => {
    navigator.clipboard.writeText(value);
    enqueueSnackbar({ message: t("user.invitationCodeCopied"), variant: "success", action: DefaultCloseAction });
  };

  const formatDate = (value?: string) => (value ? new Date(value).toLocaleString() : "-");

  return (
    <DraggableDialog
      title={t("user.invitationCodes")}
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "md",
      }}
    >
      <DialogContent>
        <Stack spacing={2}>
          <Typography variant="subtitle2" color="text.secondary">
            {t("user.invitationCodesDes")}
          </Typography>
          <Stack direction={{ xs: "column", sm: "row" }} spacing={1} alignItems={{ sm: "flex-end" }}>
            <SettingForm title={t("user.invitationCode")} lgWidth={4}>
              <DenseFilledTextField
                fullWidth
                placeholder={t("user.invitationCodeAuto")}
                value={code}
                onChange={(e) => setCode(e.target.value)}
              />
            </SettingForm>
            <SettingForm title={t("user.invitationCodeGroup")} lgWidth={3}>
              <GroupSelectionInput value={groupId} onChange={setGroupId} fullWidth />
            </SettingForm>
            <SettingForm title={t("user.invitationCodeMaxUses")} lgWidth={2}>
              <DenseFilledTextField
                fullWidth
                type="number"
                slotProps={{ htmlInput: { min: 0 } }}
                value={maxUses}
                onChange={(e) => setMaxUses(e.target.value)}
              />
            </SettingForm>
            <SettingForm title={t("user.invitationCodeExpires")} lgWidth={3}>
              <DenseFilledTextField
                fullWidth
                type="datetime-local"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
            </SettingForm>
            <Box sx={{ pb: 0.5 }}>
              <Button
                variant="contained"
                startIcon={<Add />}
                onClick={handleCreate}
                disabled={creating}
                sx={{ whiteSpace: "nowrap" }}
              >
                {t("group.create")}
              </Button>
            </Box>
          </Stack>

          <Stack direction="row" justifyContent="flex-end">
            <SecondaryButton size="small" startIcon={<ArrowSync />} onClick={() => fetchCodes()} disabled={loading}>
              {t("node.refresh")}
            </SecondaryButton>
          </Stack>

          <TableContainer component={StyledTableContainerPaper}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <NoWrapTableCell>{t("user.invitationCode")}</NoWrapTableCell>
                  <NoWrapTableCell>{t("user.group")}</NoWrapTableCell>
                  <NoWrapTableCell>{t("user.invitationCodeUses")}</NoWrapTableCell>
                  <NoWrapTableCell>{t("user.invitationCodeExpires")}</NoWrapTableCell>
                  <NoWrapTableCell>{t("user.createdAt")}</NoWrapTableCell>
                  <NoWrapTableCell align="right" />
                </TableRow>
              </TableHead>
              <TableBody>
                {codes.map((c) => {
                  const expired = !!c.expires_at && new Date(c.expires_at) < new Date();
                  const exhausted = c.max_uses > 0 && c.used_count >= c.max_uses;
                  return (
                    <TableRow key={c.id} sx={{ opacity: expired || exhausted ? 0.5 : 1 }}>
                      <TableCell>
                        <Stack direction="row" spacing={0.5} alignItems="center">
                          <Typography variant="body2" fontFamily="monospace">
                            {c.code}
                          </Typography>
                          <Tooltip title={t("user.invitationCodeCopy")}>
                            <IconButton size="small" onClick={() => handleCopy(c.code)}>
                              <CopyOutlined fontSize="inherit" />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </TableCell>
                      <TableCell>{c.group_id > 0 ? c.group_id : t("user.invitationCodeDefaultGroup")}</TableCell>
                      <TableCell>
                        {c.used_count}/{c.max_uses > 0 ? c.max_uses : "∞"}
                      </TableCell>
                      <TableCell>{formatDate(c.expires_at)}</TableCell>
                      <TableCell>{formatDate(c.created_at)}</TableCell>
                      <TableCell align="right">
                        <IconButton size="small" color="error" onClick={() => handleDelete(c.id)}>
                          <Delete fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {codes.length === 0 && !loading && (
                  <TableRow>
                    <TableCell colSpan={6} align="center">
                      <Typography variant="body2" color="text.secondary" sx={{ py: 3 }}>
                        {t("user.noInvitationCodes")}
                      </Typography>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
          {nextToken && (
            <Box sx={{ textAlign: "center" }}>
              <SecondaryButton size="small" onClick={() => fetchCodes(nextToken)} disabled={loading}>
                {t("user.loadMore")}
              </SecondaryButton>
            </Box>
          )}
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default InvitationCodeDialog;
