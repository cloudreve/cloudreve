import {
  Avatar,
  Box,
  DialogContent,
  Divider,
  Grid,
  List,
  ListItem,
  ListItemAvatar,
  ListItemButton,
  ListItemText,
  Stack,
} from "@mui/material";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Viewer, ViewerType } from "../../../api/explorer.ts";
import { closeViewerSelector } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { ViewersByID } from "../../../redux/siteConfigSlice.ts";
import { builtInViewers, openViewer } from "../../../redux/thunks/viewer.ts";
import SessionManager, { UserSettings } from "../../../session";
import { fileExtension } from "../../../util";
import AutoHeight from "../../Common/AutoHeight.tsx";
import { SecondaryButton } from "../../Common/StyledComponents.tsx";
import DraggableDialog, { StyledDialogContentText } from "../../Dialogs/DraggableDialog.tsx";
import Book from "../../Icons/Book.tsx";
import DocumentPDF from "../../Icons/DocumentPDF.tsx";
import FolderZip from "../../Icons/FolderZip.tsx";
import Image from "../../Icons/Image.tsx";
import Markdown from "../../Icons/Markdown.tsx";
import MoreHorizontal from "../../Icons/MoreHorizontal.tsx";
import MusicNote1 from "../../Icons/MusicNote1.tsx";

export interface ViewerIconProps {
  viewer: Viewer;
  size?: number;
  py?: number;
}

const emptyViewer: Viewer[] = [];

export const ViewerIDWithDefaultIcons = [
  builtInViewers.image,
  builtInViewers.pdf,
  builtInViewers.epub,
  builtInViewers.music,
  builtInViewers.markdown,
  builtInViewers.archive,
];

export const ViewerIcon = ({ viewer, size = 32, py = 0.5 }: ViewerIconProps) => {
  const BuiltinIcons = useMemo(() => {
    if (viewer.icon) {
      return undefined;
    }

    if (viewer.type == ViewerType.builtin) {
      switch (viewer.id) {
        case builtInViewers.image:
          return <Image sx={{ width: size, height: size, color: "#d32f2f" }} />;
        case builtInViewers.pdf:
          return <DocumentPDF sx={{ width: size, height: size, color: "#f44336" }} />;
        case builtInViewers.epub:
          return <Book sx={{ width: size, height: size, color: "#81b315" }} />;
        case builtInViewers.music:
          return <MusicNote1 sx={{ width: size, height: size, color: "#651fff" }} />;
        case builtInViewers.archive:
          return <FolderZip sx={{ width: size, height: size, color: "#f9a825" }} />;
        case builtInViewers.markdown:
          return (
            <Markdown
              sx={{
                width: size,
                height: size,
                color: (theme) => (theme.palette.mode == "dark" ? "#cbcbcb" : "#383838"),
              }}
            />
          );
      }
    }
  }, [viewer]);
  return (
    <Box sx={{ display: "flex", py }}>
      {BuiltinIcons && BuiltinIcons}
      {viewer.icon && (
        <Box
          component={"img"}
          src={viewer.icon}
          sx={{
            width: size,
            height: size,
          }}
        />
      )}
    </Box>
  );
};

const OpenWith = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [selectedViewer, setSelectedViewer] = React.useState<Viewer | null>(null);
  const [expanded, setExpanded] = useState(false);

  const selectorState = useAppSelector((state) => state.globalState.viewerSelector);

  useEffect(() => {
    if (selectorState?.open) {
      setExpanded(!selectorState.viewers);
      setSelectedViewer(null);
    }
  }, [selectorState]);

  const ext = useMemo(() => {
    if (!selectorState?.file) {
      return "";
    }

    return fileExtension(selectorState.file.name) ?? "";
  }, [selectorState?.file]);

  const onClose = useCallback(() => {
    dispatch(closeViewerSelector());
  }, [dispatch]);

  const openWith = (always: boolean, viewer?: Viewer) => {
    if (!selectorState || (!selectedViewer && !viewer)) {
      return;
    }

    if (always) {
      SessionManager.set(UserSettings.OpenWithPrefix + ext, viewer?.id ?? selectedViewer?.id);
    }

    dispatch(
      openViewer(
        selectorState.file,
        viewer ?? (selectedViewer as Viewer),
        selectorState.entitySize,
        selectorState.version,
      ),
    );
    dispatch(closeViewerSelector());
  };

  const onViewerClick = (viewer: Viewer) => {
    if (selectorState?.viewers) {
      setSelectedViewer(viewer);
    } else {
      // For files without matching viewers, open the selected viewer without asking for preference
      openWith(false, viewer);
    }
  };

  return (
    <DraggableDialog
      title={t("application:fileManager.openWith")}
      dialogProps={{
        open: !!(selectorState && selectorState.open),
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
      }}
    >
      <AutoHeight>
        <DialogContent sx={{ pb: selectedViewer ? 0 : 2 }}>
          <Stack spacing={2}>
            <StyledDialogContentText>
              {t("fileManager.openWithDescription", {
                ext,
              })}
            </StyledDialogContentText>
          </Stack>
          <List
            sx={{
              width: "100%",
              maxHeight: "calc(100vh - 400px)",
              overflow: "auto",
            }}
          >
            {((expanded ? Object.values(ViewersByID) : selectorState?.viewers) ?? emptyViewer).map((viewer) => (
              <ListItem
                disablePadding
                key={viewer.id}
                onDoubleClick={() => openWith(false, viewer)}
                onClick={() => onViewerClick(viewer)}
              >
                <ListItemButton selected={viewer.id == selectedViewer?.id}>
                  <ListItemAvatar sx={{ minWidth: "48px" }}>
                    <ViewerIcon viewer={viewer} />
                  </ListItemAvatar>
                  <ListItemText primary={t(viewer.display_name)} />
                </ListItemButton>
              </ListItem>
            ))}
            {!expanded && (
              <ListItem onClick={() => setExpanded(true)} disablePadding>
                <ListItemButton>
                  <ListItemAvatar sx={{ minWidth: "48px" }}>
                    <Avatar sx={{ width: 32, height: 32 }}>
                      <MoreHorizontal />
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText primary={t("fileManager.expandAllApp")} />
                </ListItemButton>
              </ListItem>
            )}
          </List>
        </DialogContent>
        {!!selectedViewer && (
          <>
            <Divider />
            <Grid container spacing={2} sx={{ p: 2 }}>
              <Grid md={6} xs={12} item>
                <SecondaryButton fullWidth variant={"contained"} onClick={() => openWith(true)}>
                  {t("modals.always")}
                </SecondaryButton>
              </Grid>
              <Grid md={6} xs={12} item>
                <SecondaryButton fullWidth variant={"contained"} onClick={() => openWith(false)}>
                  {t("modals.justOnce")}
                </SecondaryButton>
              </Grid>
            </Grid>
          </>
        )}
      </AutoHeight>
    </DraggableDialog>
  );
};
export default OpenWith;
