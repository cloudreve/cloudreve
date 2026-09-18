import { Box, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import { memo, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { DirectLink, FileType } from "../../../../api/explorer.ts";
import { copyToClipboard } from "../../../../util/index.ts";
import CopyOutlined from "../../../Icons/CopyOutlined.tsx";
import FileIcon from "../../Explorer/FileIcon.tsx";

interface DirectLinkItemProps {
  target: DirectLink;
  index: number;
  displayLink: string;
  displayName?: string;
  showDisplayName: boolean;
  relativePathPlaceholder: string;
}

const DirectLinkItem = memo(
  ({ target, index, displayLink, displayName, showDisplayName, relativePathPlaceholder }: DirectLinkItemProps) => {
    const { t } = useTranslation();
    const handleCopy = useCallback(() => {
      copyToClipboard(displayLink);
    }, [displayLink]);

    return (
      <Stack
        direction="row"
        alignItems="center"
        spacing={1}
        sx={{
          px: 1.5,
          py: 0.75,
          borderRadius: 1,
          border: "1px solid",
          borderColor: "divider",
          "&:hover": {
            bgcolor: "action.hover",
            boxShadow: (theme) => `inset 0 0 0 1px ${theme.palette.primary.light}`,
          },
          transition: (theme) => theme.transitions.create(["background-color", "box-shadow"]),
          minWidth: 0,
        }}
      >
        <FileIcon
          sx={{ px: 0, py: 0, height: "20px", flexShrink: 0 }}
          variant="small"
          iconProps={{ fontSize: "small" }}
          file={{
            type: FileType.file,
            name: displayName ?? "",
          }}
        />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          {showDisplayName && displayName && (
            <Typography variant="body2" color="text.secondary" noWrap>
              {displayName}
            </Typography>
          )}
          <Typography
            variant="body2"
            noWrap
            component="div"
            sx={{ fontFamily: "monospace", fontSize: "0.8rem" }}
            title={displayLink}
          >
            {displayLink}
          </Typography>
        </Box>
        <Tooltip title={t("application:fileManager.copy")}>
          <IconButton size="small" onClick={handleCopy} sx={{ flexShrink: 0 }}>
            <CopyOutlined fontSize="small" />
          </IconButton>
        </Tooltip>
      </Stack>
    );
  },
);

DirectLinkItem.displayName = "DirectLinkItem";

export default DirectLinkItem;
