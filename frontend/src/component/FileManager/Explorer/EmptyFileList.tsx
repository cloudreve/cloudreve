import {
  Alert,
  AlertTitle,
  Box,
  ListItemIcon,
  ListItemText,
  MenuList,
  Paper,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import { grey } from "@mui/material/colors";
import React, { memo, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { NavigatorCapability } from "../../../api/explorer.ts";
import { useAppSelector } from "../../../redux/hooks.ts";
import { isMacbook } from "../../../redux/thunks/file.ts";
import Boolset from "../../../util/boolset.ts";
import { Filesystem } from "../../../util/uri.ts";
import Nothing from "../../Common/Nothing.tsx";
import { KeyIndicator } from "../../Frame/NavBar/SearchBar.tsx";
import ArrowSync from "../../Icons/ArrowSync.tsx";
import Border from "../../Icons/Border.tsx";
import BorderAll from "../../Icons/BorderAll.tsx";
import BorderInside from "../../Icons/BorderInside.tsx";
import FolderLink from "../../Icons/FolderLink.tsx";
import MoreHorizontal from "../../Icons/MoreHorizontal.tsx";
import PinOutlined from "../../Icons/PinOutlined.tsx";
import { DenseDivider, SquareMenuItem } from "../ContextMenu/ContextMenu.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { ActionButton, ActionButtonGroup } from "../TopBar/TopActions.tsx";

interface EmptyFileListProps {
  [key: string]: any;
}

export const SearchLimitReached = () => {
  const { t } = useTranslation("application");
  return (
    <Alert severity="warning">
      <AlertTitle> {t("fileManager.recursiveLimitReached")}</AlertTitle>
      {t("fileManager.recursiveLimitReachedDes")}
    </Alert>
  );
};

export const SharedWithMeEmpty = () => {
  const { t } = useTranslation("application");

  return (
    <Stack spacing={1}>
      <Box
        sx={{
          width: "300px",
          height: "200px",
          overflow: "hidden",
          backgroundColor: (t) => (t.palette.mode == "dark" ? grey[900] : grey[100]),
          borderRadius: (t) => `${t.shape.borderRadius}px`,
          p: 1,
          position: "relative",
          "&::after": {
            content: '""',
            position: "absolute",
            bottom: 0,
            left: 0,
            width: "100%",
            height: "50px",
            background: (t) =>
              `linear-gradient(to bottom, transparent, ${t.palette.mode == "dark" ? grey[900] : grey[100]})`,
            pointerEvents: "none",
          },
        }}
      >
        <Box
          sx={{
            display: "flex",
            pointerEvents: "none",

            flexDirection: "column",
            justifyContent: "flex-end",
          }}
        >
          <Box sx={{ display: "flex", gap: 1 }}>
            <Box
              sx={{
                flexGrow: 1,
                border: (t) => `1px solid ${t.palette.divider}`,
                backgroundColor: (t) => t.palette.background.paper,
                height: "42px",
                borderRadius: (t) => `${t.shape.borderRadius}px`,
              }}
            />
            <ActionButtonGroup
              variant="outlined"
              sx={{
                backgroundColor: (t) => t.palette.background.paper,
                height: "42px",
              }}
            >
              <Tooltip enterDelay={200} title={t("application:fileManager.refresh")}>
                <ActionButton>
                  <ArrowSync fontSize={"small"} />
                </ActionButton>
              </Tooltip>
              <ActionButton
                sx={{
                  border: (t) => `1px solid ${t.palette.primary.main}`,
                }}
              >
                <MoreHorizontal fontSize={"small"} />
              </ActionButton>
            </ActionButtonGroup>
          </Box>
          <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
            <Paper elevation={3} sx={{ borderRadius: "8px" }}>
              <MenuList dense sx={{ padding: "4px 0", minWidth: "200px" }}>
                <SquareMenuItem>
                  <ListItemIcon>
                    <PinOutlined fontSize="small" />
                  </ListItemIcon>
                  <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                    {t("application:fileManager.pin")}
                  </ListItemText>
                </SquareMenuItem>
                <SquareMenuItem selected>
                  <ListItemIcon>
                    <FolderLink fontSize="small" />
                  </ListItemIcon>
                  <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                    {t("application:fileManager.saveShortcut")}
                  </ListItemText>
                </SquareMenuItem>
                <DenseDivider />
                <SquareMenuItem>
                  <ListItemIcon>
                    <PinOutlined fontSize="small" />
                  </ListItemIcon>
                  <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                    {t("application:fileManager.pin")}
                  </ListItemText>
                </SquareMenuItem>
                <SquareMenuItem>
                  <ListItemIcon>
                    <BorderAll fontSize="small" />
                  </ListItemIcon>
                  <ListItemText>{t("application:fileManager.selectAll")}</ListItemText>
                  <Typography variant="body2" color="text.secondary">
                    <KeyIndicator>{isMacbook ? "⌘" : "Ctrl"}</KeyIndicator>+<KeyIndicator>A</KeyIndicator>
                  </Typography>
                </SquareMenuItem>
                <SquareMenuItem>
                  <ListItemIcon>
                    <Border fontSize="small" />
                  </ListItemIcon>
                  <ListItemText>{t("application:fileManager.selectNone")}</ListItemText>
                </SquareMenuItem>
                <SquareMenuItem>
                  <ListItemIcon>
                    <BorderInside fontSize="small" />
                  </ListItemIcon>
                  <ListItemText>{t("application:fileManager.invertSelection")}</ListItemText>
                </SquareMenuItem>
              </MenuList>
            </Paper>
          </Box>
        </Box>
      </Box>
      <Stack spacing={1} sx={{ maxWidth: "400px" }}>
        <Typography variant="h6" fontWeight={600}>
          {t("application:fileManager.shareWithMeEmpty")}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t("application:fileManager.shareWithMeEmptyDes")}
        </Typography>
      </Stack>
    </Stack>
  );
};

const EmptyFileList = memo(
  React.forwardRef(({ ...rest }: EmptyFileListProps, ref) => {
    const { t } = useTranslation("application");
    const fmIndex = useContext(FmIndexContext);
    const currentFs = useAppSelector((state) => state.fileManager[fmIndex]?.current_fs);
    const search_params = useAppSelector((state) => state.fileManager[fmIndex]?.search_params);
    const recursion_limit_reached = useAppSelector((state) => state.fileManager[fmIndex].list?.recursion_limit_reached);
    const capability = useAppSelector((state) => state.fileManager[fmIndex].list?.props.capability);

    const canCreate = useMemo(() => {
      const bs = new Boolset(capability);
      return bs.enabled(NavigatorCapability.create_file);
    }, [capability]);

    return (
      <Box
        ref={ref}
        {...rest}
        sx={{
          p: 2,
          height: "100%",
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          flexDirection: "column",
          ...rest.sx,
        }}
      >
        {currentFs == Filesystem.shared_with_me && (
          <>
            <SharedWithMeEmpty />
            {recursion_limit_reached && <SearchLimitReached />}
          </>
        )}
        {currentFs != Filesystem.shared_with_me && (
          <>
            <Nothing
              primary={search_params || !canCreate ? t("fileManager.nothingFound") : t("fileManager.dropFileHere")}
              secondary={search_params || !canCreate ? undefined : t("fileManager.orClickUploadButton")}
            />
            {recursion_limit_reached && <SearchLimitReached />}
          </>
        )}
      </Box>
    );
  }),
);

export default EmptyFileList;
