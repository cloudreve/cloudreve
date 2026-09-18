import {
  DialogContent,
  DialogProps,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  SelectChangeEvent,
  Slider,
  Stack,
} from "@mui/material";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import DialogAccordion from "../../Dialogs/DialogAccordion.tsx";
import AutoHeight from "../../Common/AutoHeight.tsx";

export interface SaveAsNewFormatProps extends DialogProps {
  onSaveSubmit: (ext: string, quality?: number) => void;
}
export interface Format {
  ext: string;
  display: string;
  quality?: boolean;
}
const formats: Format[] = [
  {
    ext: "png",
    display: "PNG",
  },
  {
    ext: "jpg",
    display: "JPEG",
    quality: true,
  },
  {
    ext: "webp",
    display: "WebP",
    quality: true,
  },
  {
    ext: "pdf",
    display: "PDF",
  },
  {
    ext: "svg",
    display: "SVG",
  },
  {
    ext: "gif",
    display: "GIF",
  },
  {
    ext: "mp4",
    display: "MP4",
  },
  {
    ext: "dds",
    display: "DDS",
  },
  {
    ext: "tiff",
    display: "TIFF",
  },
  {
    ext: "tga",
    display: "TGA",
  },
  {
    ext: "bmp",
    display: "BMP",
  },
  {
    ext: "ico",
    display: "ICO",
  },
  {
    ext: "dxf",
    display: "DXF",
  },
  {
    ext: "raw",
    display: "RAW",
  },
  {
    ext: "emf",
    display: "EMF",
  },
  {
    ext: "ppm",
    display: "PPM",
  },
];

const SaveAsNewFormat = ({ onSaveSubmit, onClose, ...rest }: SaveAsNewFormatProps) => {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<Format>(formats[0]);
  const [quality, setQuality] = useState(0.9);

  const handleChange = (event: SelectChangeEvent) => {
    setSelected(formats.find((f) => f.ext === event.target.value) ?? formats[0]);
  };

  const handleSliderChange = (_event: Event, newValue: number | number[]) => {
    setQuality(newValue as number);
  };

  const onAccept = () => {
    onSaveSubmit(selected.ext, selected.quality ? quality : undefined);
    onClose && onClose({}, "backdropClick");
  };

  return (
    <DraggableDialog
      title={t("application:modals.saveAsOtherFormat")}
      showActions
      showCancel
      onAccept={onAccept}
      dialogProps={{
        fullWidth: true,
        onClose,
        maxWidth: "xs",
        ...rest,
      }}
    >
      <DialogContent>
        <AutoHeight>
          <Stack spacing={2}>
            <FormControl variant="filled" sx={{ width: "100%" }}>
              <InputLabel>{t("fileManager.format")}</InputLabel>
              <Select value={selected.ext} onChange={handleChange}>
                {formats.map((f) => (
                  <MenuItem key={f.ext} value={f.ext}>
                    {f.display}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            {selected.quality && (
              <DialogAccordion title={t("modals.quality")}>
                <Slider
                  size={"small"}
                  valueLabelDisplay="auto"
                  min={0.01}
                  step={0.01}
                  max={1.0}
                  value={quality}
                  onChange={handleSliderChange}
                />
              </DialogAccordion>
            )}
          </Stack>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default SaveAsNewFormat;
