import {
  Box,
  DialogContent,
  FormControlLabel,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DirectLink } from "../../../../api/explorer.ts";
import { closeDirectLinkDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import CrUri from "../../../../util/uri.ts";
import { NoLabelFilledTextField, StyledCheckbox } from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import AppsList from "../../../Icons/AppsList.tsx";
import SlideText from "../../../Icons/SlideText.tsx";
import DirectLinkItem from "./DirectLinkItem.tsx";

type ViewMode = "list" | "text";

const DirectLinks = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [showFileName, setShowFileName] = useState(false);
  const [forceDownload, setForceDownload] = useState(false);
  const [copiedAll, setCopiedAll] = useState(false);

  const open = useAppSelector((state) => state.globalState.directLinkDialogOpen);
  const targets = useAppSelector((state) => state.globalState.directLinkRes);

  const relativePathPlaceholder = t("application:modals.relativePathPlaceholder");

  const getDisplayLink = useCallback(
    (link: string) => {
      if (forceDownload) {
        return link.replace("/f/", "/f/d/");
      }
      return link;
    },
    [forceDownload],
  );

  const getDisplayName = useCallback((fileUrl: string) => {
    const crUri = new CrUri(fileUrl);
    const elements = crUri.elements();
    return elements.pop() ?? "";
  }, []);

  const textContents = useMemo(() => {
    if (!targets) return "";
    return targets
      .map((target) => {
        const finalLink = getDisplayLink(target.link);
        if (!showFileName) return `${finalLink}`;
        return `[${getDisplayName(target.file_url)}] ${finalLink}`;
      })
      .join("\n");
  }, [targets, showFileName, getDisplayLink, getDisplayName, relativePathPlaceholder]);

  const onClose = useCallback(() => {
    dispatch(closeDirectLinkDialog());
  }, [dispatch]);

  const handleViewModeChange = useCallback((_: React.MouseEvent, newMode: ViewMode | null) => {
    if (newMode !== null) {
      setViewMode(newMode);
    }
  }, []);

  return (
    <DraggableDialog
      title={t("application:modals.getSourceLinkTitle")}
      showActions
      hideOk
      secondaryAction={
        <Stack direction={isMobile ? "column" : "row"} spacing={1}>
          <FormControlLabel
            sx={{ ml: 0 }}
            slotProps={{
              typography: { variant: "body2", pl: 1, color: "text.secondary" },
            }}
            control={
              <StyledCheckbox
                onChange={() => setShowFileName(!showFileName)}
                disableRipple
                checked={showFileName}
                size="small"
              />
            }
            label={t("application:modals.showFileName")}
          />
          <FormControlLabel
            sx={{ ml: 0 }}
            slotProps={{
              typography: { variant: "body2", pl: 1, color: "text.secondary" },
            }}
            control={
              <StyledCheckbox
                onChange={() => setForceDownload(!forceDownload)}
                disableRipple
                checked={forceDownload}
                size="small"
              />
            }
            label={t("application:modals.forceDownload")}
          />
        </Stack>
      }
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent sx={{ pt: 0, pb: 0 }}>
        {(targets?.length ?? 0) > 1 && (
          <ToggleButtonGroup value={viewMode} exclusive onChange={handleViewModeChange} sx={{ mb: 1 }} size="small">
            <ToggleButton value="list">
              <AppsList fontSize="small" sx={{ mr: 0.5 }} />
              <Typography variant="caption">{t("application:fileManager.listView")}</Typography>
            </ToggleButton>
            <ToggleButton value="text">
              <SlideText fontSize="small" sx={{ mr: 0.5 }} />
              <Typography variant="caption">{t("application:modals.textView")}</Typography>
            </ToggleButton>
          </ToggleButtonGroup>
        )}

        {viewMode === "list" ? (
          <DirectLinkList
            targets={targets}
            showFileName={showFileName}
            getDisplayLink={getDisplayLink}
            getDisplayName={getDisplayName}
            relativePathPlaceholder={relativePathPlaceholder}
          />
        ) : (
          <NoLabelFilledTextField
            autoFocus
            fullSize
            multiline
            value={textContents}
            variant="outlined"
            fullWidth
            slotProps={{
              htmlInput: { readOnly: true },
            }}
          />
        )}
      </DialogContent>
    </DraggableDialog>
  );
};

interface DirectLinkListProps {
  targets?: DirectLink[];
  showFileName: boolean;
  getDisplayLink: (link: string) => string;
  getDisplayName: (fileUrl: string) => string;
  relativePathPlaceholder: string;
}

const DirectLinkList = ({
  targets,
  showFileName,
  getDisplayLink,
  getDisplayName,
  relativePathPlaceholder,
}: DirectLinkListProps) => {
  return (
    <Box sx={{ maxHeight: 400, overflow: "auto" }}>
      <Stack spacing={0.5}>
        {targets?.map((target, index) => (
          <DirectLinkItem
            key={index}
            target={target}
            showDisplayName={showFileName}
            index={index}
            displayLink={getDisplayLink(target.link)}
            displayName={getDisplayName(target.file_url)}
            relativePathPlaceholder={relativePathPlaceholder}
          />
        ))}
      </Stack>
    </Box>
  );
};

export default DirectLinks;
