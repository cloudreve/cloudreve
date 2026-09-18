import { Box, DialogContent, DialogProps, Slider, Stack, Typography, useTheme } from "@mui/material";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { useTranslation } from "react-i18next";
import { CSSProperties, useEffect, useState } from "react";
import { SubtitleStyle } from "./VideoViewer.tsx";
import SessionManager, { UserSettings } from "../../../session";
import Sketch from "@uiw/react-color-sketch";

export interface SubtitleStyleProps extends DialogProps {
  onSaveSubmit: (setting: SubtitleStyle) => void;
}

const SubtitleStyleDialog = ({ onSaveSubmit, onClose, open, ...rest }: SubtitleStyleProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const [style, setStyle] = useState<SubtitleStyle>({});

  const handleSliderChange = (_event: Event, newValue: number | number[]) => {
    setStyle((s) => ({
      ...s,
      fontSize: newValue as number,
    }));
  };

  useEffect(() => {
    if (open) {
      setStyle(SessionManager.getWithFallback(UserSettings.SubtitleStyle));
    }
  }, [open]);

  const onAccept = () => {
    onSaveSubmit(style);
    onClose && onClose({}, "backdropClick");
  };

  return (
    <DraggableDialog
      title={t("application:fileManager.subtitleStyles")}
      showActions
      showCancel
      onAccept={onAccept}
      dialogProps={{
        fullWidth: true,
        onClose,
        open,
        maxWidth: "sm",
        ...rest,
      }}
    >
      <DialogContent>
        <Stack spacing={2} direction={"row"}>
          <Box>
            <Typography variant={"body2"} gutterBottom>
              {t("fileManager.color")}
            </Typography>
            <Sketch
              presetColors={false}
              style={
                {
                  border: "none",
                  boxShadow: "none",
                  padding: 0,
                  margin: 0,
                  background: theme.palette.background.default + "!important",
                } as CSSProperties
              }
              disableAlpha={true}
              color={style.fontColor ?? "#fff"}
              onChange={(color) => {
                setStyle((s) => ({
                  ...s,
                  fontColor: color.hex,
                }));
              }}
            />
          </Box>
          <Box sx={{ flexGrow: 1 }}>
            <Typography variant={"body2"} gutterBottom>
              {t("fileManager.fontSize")}
            </Typography>
            <Slider
              size={"small"}
              valueLabelDisplay="auto"
              min={5}
              step={1}
              max={50}
              value={style.fontSize ?? 20}
              onChange={handleSliderChange}
            />
            <Box>
              <Typography
                sx={{
                  textAlign: "center",
                  textShadow:
                    "#000 1px 0 1px,#000 0 1px 1px,#000 -1px 0 1px,#000 0 -1px 1px,#000 1px 1px 1px,#000 -1px -1px 1px,#000 1px -1px 1px,#000 -1px 1px 1px",
                  fontSize: `${style.fontSize ?? 20}px`,
                  color: style.fontColor ?? "#fff",
                }}
              >
                {t("fileManager.testSubtitleStyle")}
              </Typography>
            </Box>
          </Box>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default SubtitleStyleDialog;
