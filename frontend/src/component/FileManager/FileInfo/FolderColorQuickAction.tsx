import { Box, BoxProps, Stack, styled, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useMemo, useState } from "react";
import { FileResponse, Metadata } from "../../../api/explorer.ts";
import CircleColorSelector, { customizeMagicColor } from "./ColorCircle/CircleColorSelector.tsx";
import SessionManager, { UserSettings } from "../../../session";
import { defaultColors } from "../../../constants";

const StyledBox = styled(Box)(({ theme }) => ({
  margin: `0 ${theme.spacing(0.5)}`,
  padding: `${theme.spacing(0.5)} ${theme.spacing(1)}`,
}));

export interface FolderColorQuickActionProps extends BoxProps {
  file: FileResponse;
  onColorChange: (color?: string) => void;
}

const FolderColorQuickAction = ({ file, onColorChange, ...rest }: FolderColorQuickActionProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const [hex, setHex] = useState<string>(
    (file.metadata && file.metadata[Metadata.icon_color]) ?? theme.palette.action.active,
  );
  const presetColors = useMemo(() => {
    const colors = new Set(defaultColors);

    const recentColors = SessionManager.get(UserSettings.UsedCustomizedIconColors) as string[] | undefined;

    if (recentColors) {
      recentColors.forEach((color) => {
        colors.add(color);
      });
    }

    return [...colors];
  }, []);
  return (
    <StyledBox {...rest}>
      <Stack spacing={1}>
        <Typography variant={"caption"}>{t("application:fileManager.folderColor")}</Typography>
        <CircleColorSelector
          colors={[theme.palette.action.active, ...presetColors, customizeMagicColor]}
          selectedColor={hex}
          onChange={(color) => {
            onColorChange(color == theme.palette.action.active ? undefined : color);
            setHex(color);
          }}
        />
      </Stack>
    </StyledBox>
  );
};

export default FolderColorQuickAction;
