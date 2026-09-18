import {
  DialogContent,
  FormControlLabel,
  IconButton,
  Link,
  ListItemText,
  SelectChangeEvent,
  Switch,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
  useTheme,
} from "@mui/material";
import FormControl from "@mui/material/FormControl";
import Grid from "@mui/material/Grid2";
import { useSnackbar } from "notistack";
import React, { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { DndProvider, useDrag, useDrop } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { Trans, useTranslation } from "react-i18next";
import { Viewer, ViewerPlatform, ViewerType } from "../../../../api/explorer.ts";
import { builtInViewers } from "../../../../redux/thunks/viewer.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import CircularProgress from "../../../Common/CircularProgress.tsx";
import SizeInput from "../../../Common/SizeInput.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import {
  DenseFilledTextField,
  DenseSelect,
  NoWrapTableCell,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import { ViewerIDWithDefaultIcons } from "../../../FileManager/Dialogs/OpenWith.tsx";
import Add from "../../../Icons/Add.tsx";
import ArrowDown from "../../../Icons/ArrowDown.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import MagicVarDialog, { MagicVar } from "../../Common/MagicVarDialog.tsx";
import { NoMarginHelperText } from "../../Settings/Settings.tsx";

const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor.tsx"));

export interface FileViewerEditDialogProps {
  viewer: Viewer;
  onChange: (viewer: Viewer) => void;
  open: boolean;
  onClose: () => void;
}

const magicVars: MagicVar[] = [
  {
    name: "{$src}",
    value: "settings.srcEncodedVar",
    example: "https%3A%2F%2Fcloudreve.org%2Fapi%2Fv4%2Ffile%2Fcontent%2FzOie%2F0%2Ftext.txt%3Fsign%3Dxxx",
  },
  {
    name: "{$src_raw}",
    value: "settings.srcVar",
    example: "https://cloudreve.org/api/v4/file/content/zOie/0/text.txt?sign=xxx",
  },
  {
    name: "{$src_raw_base64}",
    value: "settings.srcBase64Var",
    example: "aHR0cHM6Ly9jbG91ZHJldmUub3JnL2FwaS92NC9maWxlL2NvbnRlbnQvek9pZS8wL3RleHQudHh0P3NpZ249eHh4",
  },
  {
    name: "{$name}",
    value: "settings.nameEncodedVar",
    example: "sampleFile%5B1%5D.txt",
  },
  {
    name: "{$version}",
    value: "settings.versionEntityVar",
    example: "zOie",
  },
  {
    name: "{$id}",
    value: "settings.fileIdVar",
    example: "jm8AF8",
  },
  {
    name: "{$user_id}",
    value: "settings.userIdVar",
    example: "lpua",
  },
  {
    name: "{$user_display_name}",
    value: "settings.userDisplayNameVar",
    example: "Aaron%20Liu",
  },
  {
    name: "{$theme}",
    value: "settings.themeVar",
    example: "dark",
  },
  {
    name: "{$dark}",
    value: "settings.darkVar",
    example: "1",
  },
];

const DND_TYPE = "template-row";

interface DraggableTemplateRowProps {
  t: any;
  i: number;
  moveRow: (from: number, to: number) => void;
  onExtChange: (e: SelectChangeEvent<unknown>, child: React.ReactNode) => void;
  onNameChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onDelete: () => void;
  isFirst: boolean;
  isLast: boolean;
  extList: string[];
  template: any;
}

function DraggableTemplateRow({
  i,
  moveRow,
  onExtChange,
  onNameChange,
  onDelete,
  isFirst,
  isLast,
  extList,
  template,
}: DraggableTemplateRowProps) {
  const ref = React.useRef<HTMLTableRowElement>(null);
  const [, drop] = useDrop({
    accept: DND_TYPE,
    hover(item: any, monitor) {
      if (!ref.current) return;

      const dragIndex = item.index;
      const hoverIndex = i;
      if (dragIndex === hoverIndex) return;

      const hoverBoundingRect = ref.current.getBoundingClientRect();
      const hoverMiddleY = (hoverBoundingRect.bottom - hoverBoundingRect.top) / 2;
      const clientOffset = monitor.getClientOffset();
      if (!clientOffset) return;

      const hoverClientY = clientOffset.y - hoverBoundingRect.top;

      if (dragIndex < hoverIndex && hoverClientY < hoverMiddleY) return;
      if (dragIndex > hoverIndex && hoverClientY > hoverMiddleY) return;

      moveRow(dragIndex, hoverIndex);
      item.index = hoverIndex;
    },
  });
  const [{ isDragging }, drag] = useDrag({
    type: DND_TYPE,
    item: { index: i },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });
  drag(drop(ref));
  return (
    <TableRow
      ref={ref}
      sx={{ "&:last-child td, &:last-child th": { border: 0 }, opacity: isDragging ? 0.5 : 1, cursor: "move" }}
      hover
    >
      <NoWrapTableCell>
        <DenseSelect value={template.ext} required onChange={onExtChange}>
          {extList.map((ext) => (
            <SquareMenuItem value={ext} key={ext}>
              <ListItemText slotProps={{ primary: { variant: "body2" } }}>{ext}</ListItemText>
            </SquareMenuItem>
          ))}
        </DenseSelect>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <DenseFilledTextField fullWidth required value={template.display_name} onChange={onNameChange} />
      </NoWrapTableCell>
      <NoWrapTableCell>
        <IconButton size={"small"} onClick={onDelete}>
          <Dismiss fontSize={"small"} />
        </IconButton>
        <IconButton size="small" onClick={() => moveRow(i, i - 1)} disabled={isFirst}>
          <ArrowDown
            sx={{
              width: "18px",
              height: "18px",
              transform: "rotate(180deg)",
            }}
          />
        </IconButton>
        <IconButton size="small" onClick={() => moveRow(i, i + 1)} disabled={isLast}>
          <ArrowDown
            sx={{
              width: "18px",
              height: "18px",
            }}
          />
        </IconButton>
      </NoWrapTableCell>
    </TableRow>
  );
}

const FileViewerEditDialog = ({ viewer, onChange, open, onClose }: FileViewerEditDialogProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const { enqueueSnackbar } = useSnackbar();
  const [viewerShadowed, setViewerShadowed] = useState<Viewer | undefined>(undefined);
  const formRef = React.useRef<HTMLFormElement>(null);
  const [magicVarOpen, setMagicVarOpen] = useState(false);
  const [wopiCached, setWopiCached] = useState("");
  const withDefaultIcon = useMemo(() => {
    return ViewerIDWithDefaultIcons.includes(viewer.id);
  }, [viewer.id]);

  useEffect(() => {
    setViewerShadowed({ ...viewer });
    setWopiCached("");
  }, [viewer, setWopiCached, setViewerShadowed]);

  const onSubmit = useCallback(() => {
    if (formRef.current && !formRef.current.checkValidity()) {
      formRef.current.reportValidity();
      return;
    }

    let changed = viewerShadowed;
    if (!viewerShadowed || !changed) {
      return;
    }

    if (wopiCached != "") {
      try {
        const parsed = JSON.parse(wopiCached);
        changed = { ...viewerShadowed, wopi_actions: parsed };
        setViewerShadowed({ ...changed });
      } catch (e) {
        enqueueSnackbar({
          message: t("settings.invalidWopiActionMapping"),
          variant: "warning",
          action: DefaultCloseAction,
        });
        return;
      }
    }

    onChange(changed);
    onClose();
  }, [viewerShadowed, wopiCached, formRef]);

  const openMagicVar = useCallback((e: React.MouseEvent<HTMLElement>) => {
    setMagicVarOpen(true);
    e.stopPropagation();
    e.preventDefault();
  }, []);

  if (!viewerShadowed) {
    return null;
  }

  return (
    <DraggableDialog
      title={t("settings.editViewerTitle", {
        name: t(viewer.display_name, { ns: "application" }),
      })}
      showActions
      showCancel
      onAccept={onSubmit}
      dialogProps={{
        fullWidth: true,
        maxWidth: "lg",
        open,
        onClose,
      }}
    >
      <DialogContent>
        <MagicVarDialog open={magicVarOpen} vars={magicVars} onClose={() => setMagicVarOpen(false)} />
        <form ref={formRef}>
          <Grid spacing={2} container>
            <SettingForm noContainer lgWidth={6} title={t("settings.iconUrl")}>
              <DenseFilledTextField
                fullWidth
                required={!withDefaultIcon}
                value={viewerShadowed.icon}
                onChange={(e) => {
                  setViewerShadowed((v) => ({
                    ...(v as Viewer),
                    icon: e.target.value,
                  }));
                }}
              />
              {withDefaultIcon && <NoMarginHelperText>{t("settings.builtInIconUrlDes")}</NoMarginHelperText>}
            </SettingForm>
            <SettingForm noContainer lgWidth={6} title={t("settings.displayName")}>
              <DenseFilledTextField
                fullWidth
                required
                value={viewerShadowed.display_name}
                onChange={(e) => {
                  setViewerShadowed((v) => ({
                    ...(v as Viewer),
                    display_name: e.target.value,
                  }));
                }}
              />
              <NoMarginHelperText>{t("settings.displayNameDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm noContainer lgWidth={6} title={t("settings.exts")}>
              <DenseFilledTextField
                fullWidth
                multiline
                required
                value={viewerShadowed.exts.join()}
                onChange={(e) =>
                  setViewerShadowed((v) => ({
                    ...(v as Viewer),
                    exts: e.target.value.split(",").map((ext) => ext.trim()),
                  }))
                }
              />
            </SettingForm>
            {viewer.type == ViewerType.custom && (
              <SettingForm noContainer lgWidth={6} title={t("settings.viewerUrl")}>
                <DenseFilledTextField
                  fullWidth
                  required
                  value={viewerShadowed.url}
                  onChange={(e) =>
                    setViewerShadowed((v) => ({
                      ...(v as Viewer),
                      url: e.target.value,
                    }))
                  }
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey={"settings.viewerUrlDes"}
                    ns={"dashboard"}
                    components={[<Link onClick={openMagicVar} href={"#"} />]}
                  />
                </NoMarginHelperText>
              </SettingForm>
            )}
            <SettingForm noContainer lgWidth={6} title={t("settings.maxSize")}>
              <FormControl fullWidth>
                <SizeInput
                  variant={"outlined"}
                  required
                  allowZero={true}
                  value={viewerShadowed.max_size ?? 0}
                  onChange={(e) =>
                    setViewerShadowed((v) => ({
                      ...(v as Viewer),
                      max_size: e ? e : undefined,
                    }))
                  }
                />
                <NoMarginHelperText>{t("settings.maxSizeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm noContainer lgWidth={6} title={t("settings.viewerPlatform")}>
              <FormControl fullWidth>
                <DenseSelect
                  value={viewerShadowed.platform || ViewerPlatform.all}
                  onChange={(e) =>
                    setViewerShadowed((v) => ({
                      ...(v as Viewer),
                      platform: e.target.value as ViewerPlatform,
                    }))
                  }
                >
                  <SquareMenuItem value="pc">
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.viewerPlatformPC")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value="mobile">
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.viewerPlatformMobile")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value="all">
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.viewerPlatformAll")}
                    </ListItemText>
                  </SquareMenuItem>
                </DenseSelect>
                <NoMarginHelperText>{t("settings.viewerPlatformDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm noContainer lgWidth={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={isTrueVal(viewerShadowed.props?.openInNew ?? "")}
                    onChange={(e) =>
                      setViewerShadowed((v) => ({
                        ...(v as Viewer),
                        props: {
                          ...(v?.props ?? {}),
                          openInNew: e.target.checked.toString(),
                        },
                      }))
                    }
                  />
                }
                label={t("settings.openInNew")}
              />
              <NoMarginHelperText>{t("settings.openInNewDes")}</NoMarginHelperText>
            </SettingForm>
            {viewer.id == builtInViewers.drawio && (
              <SettingForm noContainer title={t("settings.drawioHost")} lgWidth={6}>
                <DenseFilledTextField
                  fullWidth
                  required
                  value={viewerShadowed.props?.host ?? ""}
                  onChange={(e) =>
                    setViewerShadowed((v) => ({
                      ...(v as Viewer),
                      props: {
                        ...(v?.props ?? {}),
                        host: e.target.value,
                      },
                    }))
                  }
                />
                <NoMarginHelperText>{t("settings.drawioHostDes")}</NoMarginHelperText>
              </SettingForm>
            )}
            {viewer.type == ViewerType.wopi && (
              <SettingForm noContainer title={t("settings.woapiActionMapping")} lgWidth={12}>
                <Suspense fallback={<CircularProgress />}>
                  <MonacoEditor
                    theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                    value={wopiCached == "" ? JSON.stringify(viewerShadowed.wopi_actions, null, 4) : wopiCached}
                    height={"300px"}
                    minHeight={"300px"}
                    language={"json"}
                    onChange={(e) => setWopiCached(e as string)}
                  />
                </Suspense>
              </SettingForm>
            )}
            <SettingForm noContainer title={t("settings.newFileAction")} lgWidth={12}>
              {viewerShadowed?.templates && viewerShadowed.templates.length > 0 && (
                <DndProvider backend={HTML5Backend}>
                  <TableContainer sx={{ mt: 1, maxHeight: 440 }} component={StyledTableContainerPaper}>
                    <Table
                      stickyHeader
                      sx={{
                        width: "100%",
                        maxHeight: 300,
                        tableLayout: "fixed",
                      }}
                      size="small"
                    >
                      <TableHead>
                        <TableRow>
                          <NoWrapTableCell width={100}>{t("settings.ext")}</NoWrapTableCell>
                          <NoWrapTableCell width={200}>{t("settings.displayName")}</NoWrapTableCell>
                          <NoWrapTableCell width={100}>{t("settings.actions")}</NoWrapTableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {viewerShadowed.templates?.map((template, i) => (
                          <DraggableTemplateRow
                            key={i}
                            t={template}
                            i={i}
                            moveRow={(from, to) => {
                              if (from === to || to < 0 || to >= (viewerShadowed.templates?.length ?? 0)) return;
                              setViewerShadowed((v) => {
                                const arr = [...(v?.templates ?? [])];
                                const [moved] = arr.splice(from, 1);
                                arr.splice(to, 0, moved);
                                return { ...(v as Viewer), templates: arr };
                              });
                            }}
                            onExtChange={(e) => {
                              const newExt = e.target.value as string;
                              setViewerShadowed((v) => ({
                                ...(v as Viewer),
                                templates: (v?.templates ?? []).map((template, index) =>
                                  index == i ? { ...template, ext: newExt } : template,
                                ),
                              }));
                            }}
                            onNameChange={(e) => {
                              setViewerShadowed((v) => ({
                                ...(v as Viewer),
                                templates: (v?.templates ?? []).map((template, index) =>
                                  index == i ? { ...template, display_name: e.target.value } : template,
                                ),
                              }));
                            }}
                            onDelete={() => {
                              setViewerShadowed((v) => ({
                                ...(v as Viewer),
                                templates: (v?.templates ?? []).filter((_, index) => index != i),
                              }));
                            }}
                            isFirst={i === 0}
                            isLast={i === (viewerShadowed.templates?.length ?? 0) - 1}
                            extList={viewerShadowed.exts}
                            template={template}
                          />
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </DndProvider>
              )}
              <SecondaryButton
                sx={{ mt: 1 }}
                variant={"contained"}
                startIcon={<Add />}
                onClick={() =>
                  setViewerShadowed((v) => ({
                    ...(v as Viewer),
                    templates: [...(v?.templates ?? []), { ext: viewerShadowed.exts?.[0] ?? "", display_name: "" }],
                  }))
                }
              >
                {t("settings.addNewFileAction")}
              </SecondaryButton>
              <NoMarginHelperText>{t("settings.newFileActionDes")}</NoMarginHelperText>
            </SettingForm>
          </Grid>
        </form>
      </DialogContent>
    </DraggableDialog>
  );
};

export default FileViewerEditDialog;
