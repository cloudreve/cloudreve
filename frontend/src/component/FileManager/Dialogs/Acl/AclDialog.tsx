import {
  Box,
  Checkbox,
  DialogContent,
  IconButton,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { deleteAclEntry, getAclEntries, searchAclSubjects, upsertAclEntry } from "../../../../api/api.ts";
import { AclEntry, AclPermissionKey, AclSubject, AclSubjectType } from "../../../../api/explorer.ts";
import { closeAclDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import {
  DenseAutocomplete,
  DenseFilledTextField,
  NoWrapTableCell,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Globe from "../../../Icons/Globe.tsx";
import PeopleTeam from "../../../Icons/PeopleTeam.tsx";
import PersonOutlined from "../../../Icons/PersonOutlined.tsx";
import PersonStar from "../../../Icons/PersonStar.tsx";

const permissionOrder: AclPermissionKey[] = ["read", "create", "update", "delete"];

interface PendingSubject {
  type: AclSubjectType;
  id: number;
  label: string;
}

const AclDialog = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [entries, setEntries] = useState<AclEntry[] | undefined>(undefined);
  const [pending, setPending] = useState<number[]>([]);
  const [subjectOptions, setSubjectOptions] = useState<PendingSubject[]>([]);
  const [keyword, setKeyword] = useState("");

  const open = useAppSelector((state) => state.globalState.aclDialogOpen);
  const target = useAppSelector((state) => state.globalState.aclDialogFile);

  const uri = target?.path;

  const generalSubject = useCallback(
    (type: "anonymous" | "everyone"): PendingSubject => ({
      type,
      id: 0,
      label: t(`fileManager.acl${type === "anonymous" ? "Anonymous" : "Everyone"}`),
    }),
    [t],
  );

  const subjectLabel = useCallback(
    (e: AclEntry) => {
      if (e.subject_type === "anonymous" || e.subject_type === "everyone") {
        return t(`fileManager.acl${e.subject_type === "anonymous" ? "Anonymous" : "Everyone"}`);
      }
      return e.subject_name || `#${e.subject_id}`;
    },
    [t],
  );

  useEffect(() => {
    if (open && uri) {
      setEntries(undefined);
      dispatch(getAclEntries(uri)).then((res) => setEntries(res));
    }
  }, [open, uri, dispatch]);

  // Search selectable subjects; anonymous/everyone are always offered when not
  // already present on the file.
  useEffect(() => {
    if (!open) {
      return;
    }
    const handle = setTimeout(() => {
      const fixed: PendingSubject[] = ["anonymous", "everyone"]
        .filter((ty) => !entries?.some((e) => e.subject_type === ty))
        .map((ty) => generalSubject(ty as "anonymous" | "everyone"))
        .filter((s) => keyword == "" || s.label.toLowerCase().includes(keyword.toLowerCase()));
      dispatch(searchAclSubjects(keyword))
        .then((res: AclSubject[]) => {
          const dynamic = (res || [])
            .filter((s) => !entries?.some((e) => e.subject_type === s.type && e.subject_id === s.id))
            .map((s) => ({
              type: s.type as AclSubjectType,
              id: s.id,
              label: s.type === "user" ? s.name : `${s.name} (${t("fileManager.aclGroup")})`,
            }));
          setSubjectOptions([...dynamic, ...fixed]);
        })
        .catch(() => setSubjectOptions(fixed));
    }, 300);
    return () => clearTimeout(handle);
  }, [keyword, open, entries, dispatch, generalSubject, t]);

  const togglePermission = useCallback(
    (entry: AclEntry, perm: AclPermissionKey) => {
      if (!uri) {
        return;
      }
      const next = entry.permissions.includes(perm)
        ? entry.permissions.filter((p) => p !== perm)
        : [...entry.permissions, perm];
      setPending((p) => [...p, entry.id]);
      dispatch(
        upsertAclEntry({
          uri,
          subject_type: entry.subject_type,
          subject_id: entry.subject_id,
          permissions: next,
        }),
      )
        .then((res) => {
          setEntries((prev) =>
            prev?.map((e) => (e.id === entry.id ? { ...e, permissions: res.permissions } : e)),
          );
        })
        .finally(() => setPending((p) => p.filter((id) => id !== entry.id)));
    },
    [uri, dispatch],
  );

  const removeEntry = useCallback(
    (entry: AclEntry) => {
      if (!uri) {
        return;
      }
      setPending((p) => [...p, entry.id]);
      dispatch(deleteAclEntry(uri, entry.id))
        .then(() => setEntries((prev) => prev?.filter((e) => e.id !== entry.id)))
        .finally(() => setPending((p) => p.filter((id) => id !== entry.id)));
    },
    [uri, dispatch],
  );

  const addSubject = useCallback(
    (subject: PendingSubject | null) => {
      if (!subject || !uri) {
        return;
      }
      setPending((p) => [...p, -1]);
      dispatch(
        upsertAclEntry({
          uri,
          subject_type: subject.type,
          subject_id: subject.id,
          permissions: ["read"],
        }),
      )
        .then((res) => {
          setEntries((prev) => [...(prev ?? []), { ...res, subject_name: subject.label }]);
          setKeyword("");
        })
        .finally(() => setPending((p) => p.filter((id) => id !== -1)));
    },
    [uri, dispatch],
  );

  const subjectIcon = useMemo(
    () => ({
      user: <PersonOutlined fontSize="small" />,
      group: <PeopleTeam fontSize="small" />,
      anonymous: <PersonStar fontSize="small" />,
      everyone: <Globe fontSize="small" />,
    }),
    [],
  );

  return (
    <DraggableDialog
      title={t("application:fileManager.permissions")}
      loading={entries === undefined}
      dialogProps={{
        open: open ?? false,
        onClose: () => dispatch(closeAclDialog()),
        fullWidth: true,
        maxWidth: "md",
      }}
    >
      <DialogContent>
        <AutoHeight>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {t("fileManager.aclDes", { name: target?.name })}
          </Typography>
          <DenseAutocomplete
            options={subjectOptions}
            inputValue={keyword}
            onInputChange={(_, v) => setKeyword(v)}
            getOptionLabel={(o) => (o as PendingSubject).label}
            onChange={(_, v) => addSubject(v as PendingSubject | null)}
            value={null}
            blurOnSelect
            sx={{ mb: 2 }}
            renderOption={(props, option) => {
              const o = option as PendingSubject;
              return (
                <li {...props} key={`${o.type}:${o.id}`}>
                  <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                    {subjectIcon[o.type]}
                    {o.label}
                  </Box>
                </li>
              );
            }}
            renderInput={(params) => (
              <DenseFilledTextField
                {...params}
                placeholder={t("application:fileManager.typeToSearch")}
                variant="outlined"
                size="small"
              />
            )}
          />
          <TableContainer component={StyledTableContainerPaper}>
            <Table sx={{ width: "100%" }} size="small">
              <TableHead>
                <TableRow>
                  <NoWrapTableCell>{t("fileManager.aclSubject")}</NoWrapTableCell>
                  {permissionOrder.map((p) => (
                    <TableCell key={p} align="center" padding="checkbox">
                      {t(`fileManager.aclPerm_${p}`)}
                    </TableCell>
                  ))}
                  <TableCell padding="checkbox" />
                </TableRow>
              </TableHead>
              <TableBody>
                {entries === undefined && (
                  <TableRow>
                    <NoWrapTableCell>
                      <Skeleton variant="text" width={140} />
                    </NoWrapTableCell>
                    {permissionOrder.map((p) => (
                      <TableCell key={p} align="center" padding="checkbox">
                        <Skeleton variant="circular" width={18} height={18} sx={{ mx: "auto" }} />
                      </TableCell>
                    ))}
                    <TableCell padding="checkbox" />
                  </TableRow>
                )}
                {entries?.map((e) => (
                  <TableRow key={e.id} hover>
                    <NoWrapTableCell component="th" scope="row">
                      <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                        {subjectIcon[e.subject_type]}
                        {subjectLabel(e)}
                      </Box>
                    </NoWrapTableCell>
                    {permissionOrder.map((p) => (
                      <TableCell key={p} align="center" padding="checkbox">
                        <Checkbox
                          size="small"
                          checked={e.permissions.includes(p)}
                          disabled={pending.includes(e.id)}
                          onChange={() => togglePermission(e, p)}
                        />
                      </TableCell>
                    ))}
                    <TableCell padding="checkbox" align="right">
                      <IconButton
                        size="small"
                        disabled={pending.includes(e.id)}
                        onClick={() => removeEntry(e)}
                      >
                        <Dismiss fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {entries && entries.length === 0 && (
              <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
                <Typography variant="caption" color="text.secondary">
                  {t("application:setting.listEmpty")}
                </Typography>
              </Box>
            )}
          </TableContainer>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default AclDialog;
