import {
  Box,
  IconButton,
  ListItemText,
  Menu,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
} from "@mui/material";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { memo, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Viewer, ViewerGroup } from "../../../../api/explorer.ts";
import { DenseSelect, NoWrapTableCell, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import Add from "../../../Icons/Add.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";

// Map of extension -> viewer ID
export interface DefaultViewerAppsConfig {
  [ext: string]: string;
}

interface DefaultViewerAppsProps {
  config: string; // JSON string of DefaultViewerAppsConfig
  fileViewers: string; // JSON string of ViewerGroup[]
  onChange: (value: string) => void;
}

const DefaultViewerApps = memo(({ config, fileViewers, onChange }: DefaultViewerAppsProps) => {
  const { t } = useTranslation("dashboard");
  const { t: tApp } = useTranslation("application");
  const addPopupState = usePopupState({
    variant: "popover",
    popupId: "addDefaultViewerExt",
  });

  const parsed: DefaultViewerAppsConfig = useMemo(() => {
    try {
      return JSON.parse(config || "{}");
    } catch {
      return {};
    }
  }, [config]);

  // Build a map of ext -> available viewers (only extensions with 2+ viewers)
  const extViewerMap: { [ext: string]: Viewer[] } = useMemo(() => {
    const extMap: { [ext: string]: Viewer[] } = {};
    try {
      const groups: ViewerGroup[] = JSON.parse(fileViewers || "[]");
      groups.forEach((group) => {
        group.viewers.forEach((viewer) => {
          if (viewer.disabled) return;
          viewer.exts.forEach((ext) => {
            if (!extMap[ext]) {
              extMap[ext] = [];
            }
            extMap[ext].push(viewer);
          });
        });
      });
    } catch {
      // ignore
    }
    return extMap;
  }, [fileViewers]);

  // Extensions available to add (have 2+ viewers and not yet configured)
  const availableExts = useMemo(() => {
    return Object.entries(extViewerMap)
      .filter(([ext, viewers]) => viewers.length > 1 && !parsed[ext])
      .map(([ext]) => ext)
      .sort();
  }, [extViewerMap, parsed]);

  // Configured mappings (only those whose ext still has valid viewers)
  const configuredMappings = useMemo(() => {
    return Object.entries(parsed)
      .filter(([ext]) => extViewerMap[ext]?.length > 1)
      .sort(([a], [b]) => a.localeCompare(b));
  }, [parsed, extViewerMap]);

  const handleChange = useCallback(
    (ext: string, viewerId: string) => {
      const updated = { ...parsed };
      updated[ext] = viewerId;
      onChange(JSON.stringify(updated));
    },
    [parsed, onChange],
  );

  const handleDelete = useCallback(
    (ext: string) => {
      const updated = { ...parsed };
      delete updated[ext];
      onChange(JSON.stringify(updated));
    },
    [parsed, onChange],
  );

  const { onClose, ...menuProps } = bindMenu(addPopupState);

  const handleAddExt = useCallback(
    (ext: string) => {
      const viewers = extViewerMap[ext];
      if (!viewers?.length) return;
      const updated = { ...parsed, [ext]: viewers[0].id };
      onChange(JSON.stringify(updated));
      onClose();
    },
    [extViewerMap, parsed, onChange, onClose],
  );

  return (
    <Box>
      <SecondaryButton
        variant="contained"
        startIcon={<Add />}
        sx={{ mb: 1 }}
        disabled={availableExts.length === 0}
        {...bindTrigger(addPopupState)}
      >
        {t("settings.addDefaultViewerMapping")}
      </SecondaryButton>
      {configuredMappings.length > 0 && (
        <TableContainer sx={{ maxHeight: 440 }}>
          <Table stickyHeader sx={{ width: "100%", tableLayout: "fixed" }} size="small">
            <TableHead>
              <TableRow>
                <NoWrapTableCell width={120}>{t("settings.ext")}</NoWrapTableCell>
                <NoWrapTableCell>{t("settings.defaultViewerApp")}</NoWrapTableCell>
                <NoWrapTableCell width={64} />
              </TableRow>
            </TableHead>
            <TableBody>
              {configuredMappings.map(([ext, viewerId]) => {
                const viewers = extViewerMap[ext] ?? [];
                return (
                  <TableRow key={ext}>
                    <NoWrapTableCell>
                      <Box component="span" sx={{ fontFamily: "monospace" }}>
                        .{ext}
                      </Box>
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <DenseSelect
                        size="small"
                        fullWidth
                        value={viewers.some((v) => v.id === viewerId) ? viewerId : ""}
                        onChange={(e) => handleChange(ext, e.target.value as string)}
                      >
                        {viewers.map((viewer) => (
                          <SquareMenuItem key={viewer.id} value={viewer.id}>
                            <ListItemText
                              slotProps={{
                                primary: { variant: "body2" },
                              }}
                            >
                              {tApp(viewer.display_name, { defaultValue: viewer.display_name })}
                            </ListItemText>
                          </SquareMenuItem>
                        ))}
                      </DenseSelect>
                    </NoWrapTableCell>
                    <NoWrapTableCell>
                      <IconButton size="small" onClick={() => handleDelete(ext)}>
                        <Dismiss fontSize="small" />
                      </IconButton>
                    </NoWrapTableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <Menu
        onClose={onClose}
        anchorOrigin={{
          vertical: "top",
          horizontal: "left",
        }}
        transformOrigin={{
          vertical: "bottom",
          horizontal: "left",
        }}
        {...menuProps}
      >
        {availableExts.map((ext) => (
          <SquareMenuItem key={ext} dense onClick={() => handleAddExt(ext)}>
            .{ext}
          </SquareMenuItem>
        ))}
      </Menu>
    </Box>
  );
});

export default DefaultViewerApps;
