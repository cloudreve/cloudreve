import {
  Box,
  DialogContent,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  useTheme,
} from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { enqueueSnackbar } from "notistack";
import { getUserTags, sendDeleteTag, sendPatchTag } from "../../../api/api.ts";
import { UserTag } from "../../../api/explorer.ts";
import { defaultColors } from "../../../constants";
import { useAppDispatch } from "../../../redux/hooks.ts";
import SessionManager, { UserSettings } from "../../../session";
import { addRecentUsedColor } from "../../../session/utils.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import { FilledTextField } from "../../Common/StyledComponents.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Admin/Settings/Settings.tsx";
import CircleColorSelector, { customizeMagicColor } from "../../FileManager/FileInfo/ColorCircle/CircleColorSelector.tsx";
import FileTag from "../../FileManager/Explorer/FileTag.tsx";
import Delete from "../../Icons/Delete.tsx";
import Edit from "../../Icons/Edit.tsx";

const TagSetting = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();

  const [tags, setTags] = useState<UserTag[] | undefined>(undefined);
  const [editTarget, setEditTarget] = useState<UserTag | undefined>(undefined);
  const [deleteTarget, setDeleteTarget] = useState<UserTag | undefined>(undefined);
  const [newName, setNewName] = useState("");
  const [hex, setHex] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(false);

  const load = useCallback(() => {
    dispatch(getUserTags()).then((res) => setTags(res ?? []));
  }, [dispatch]);

  useEffect(() => {
    load();
  }, [load]);

  const presetColors = useMemo(() => {
    const colors = new Set(defaultColors);
    const recentColors = SessionManager.get(UserSettings.UsedCustomizedTagColors) as string[] | undefined;
    recentColors?.forEach((color) => colors.add(color));
    return [...colors];
  }, [hex]);

  const openEdit = useCallback((tag: UserTag) => {
    setEditTarget(tag);
    setNewName(tag.name);
    setHex(tag.color || undefined);
  }, []);

  const onColorChange = useCallback(
    (color: string | undefined) => {
      color = color == theme.palette.action.selected ? undefined : color;
      addRecentUsedColor(color, UserSettings.UsedCustomizedTagColors);
      setHex(color);
    },
    [theme],
  );

  const submitEdit = useCallback(() => {
    if (!editTarget || loading) {
      return;
    }
    const trimmed = newName.trim();
    if (!trimmed || trimmed.includes(":")) {
      enqueueSnackbar(t("application:setting.tagNameInvalid"), { variant: "error" });
      return;
    }
    setLoading(true);
    dispatch(
      sendPatchTag({
        name: editTarget.name,
        new_name: trimmed == editTarget.name ? undefined : trimmed,
        color: hex,
      }),
    )
      .then(() => {
        enqueueSnackbar(t("application:setting.tagUpdated"), { variant: "success" });
        setEditTarget(undefined);
        load();
      })
      .finally(() => setLoading(false));
  }, [dispatch, editTarget, newName, hex, loading, load, t]);

  const submitDelete = useCallback(() => {
    if (!deleteTarget || loading) {
      return;
    }
    setLoading(true);
    dispatch(sendDeleteTag(deleteTarget.name))
      .then(() => {
        enqueueSnackbar(t("application:setting.tagDeleted"), { variant: "success" });
        setDeleteTarget(undefined);
        load();
      })
      .finally(() => setLoading(false));
  }, [dispatch, deleteTarget, loading, load, t]);

  if (!tags) {
    return (
      <Box sx={{ pt: 20, display: "flex", justifyContent: "center" }}>
        <FacebookCircularProgress />
      </Box>
    );
  }

  return (
    <Stack spacing={5}>
      <SettingSection>
        <Typography variant="h6" gutterBottom>
          {t("application:setting.tags")}
        </Typography>
        <SettingSectionContent>
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{t("application:setting.tagName")}</TableCell>
                  <TableCell>{t("application:setting.tagFiles")}</TableCell>
                  <TableCell align="right">{t("application:setting.tagActions")}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tags.map((tag) => (
                  <TableRow key={tag.name}>
                    <TableCell>
                      <FileTag
                        defaultStyle
                        label={tag.name}
                        tagColor={tag.color || theme.palette.action.selected}
                        disableClick
                      />
                    </TableCell>
                    <TableCell>{tag.file_count}</TableCell>
                    <TableCell align="right">
                      <IconButton size="small" onClick={() => openEdit(tag)}>
                        <Edit fontSize="small" />
                      </IconButton>
                      <IconButton size="small" onClick={() => setDeleteTarget(tag)}>
                        <Delete fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
                {tags.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} align="center">
                      {t("application:setting.noTags")}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
          <NoMarginHelperText>{t("application:setting.tagsDes")}</NoMarginHelperText>
        </SettingSectionContent>
      </SettingSection>

      <DraggableDialog
        title={t("application:setting.editTag")}
        loading={loading}
        disabled={!newName.trim()}
        showActions
        showCancel
        onAccept={submitEdit}
        dialogProps={{
          open: editTarget != undefined,
          onClose: () => !loading && setEditTarget(undefined),
          fullWidth: true,
          maxWidth: "xs",
        }}
      >
        <DialogContent>
          <FilledTextField
            autoFocus
            margin="dense"
            label={t("application:setting.tagName")}
            type="text"
            fullWidth
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
          <Box sx={{ mt: 2 }}>
            <CircleColorSelector
              colors={[theme.palette.action.selected, ...presetColors, customizeMagicColor]}
              selectedColor={hex ?? theme.palette.action.selected}
              onChange={onColorChange}
            />
          </Box>
        </DialogContent>
      </DraggableDialog>

      <DraggableDialog
        title={t("application:setting.deleteTag")}
        loading={loading}
        showActions
        showCancel
        onAccept={submitDelete}
        dialogProps={{
          open: deleteTarget != undefined,
          onClose: () => !loading && setDeleteTarget(undefined),
          maxWidth: "xs",
        }}
      >
        <DialogContent>
          <Typography>
            {t("application:setting.deleteTagDes", { name: deleteTarget?.name, count: deleteTarget?.file_count })}
          </Typography>
        </DialogContent>
      </DraggableDialog>
    </Stack>
  );
};

export default TagSetting;
