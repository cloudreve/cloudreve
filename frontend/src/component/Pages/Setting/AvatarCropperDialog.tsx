import { DialogContent } from "@mui/material";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { useTranslation } from "react-i18next";
import { DependencyList, useEffect, useRef, useState } from "react";
import { centerCrop, Crop, makeAspectCrop, PixelCrop, ReactCrop } from "react-image-crop";
import "react-image-crop/dist/ReactCrop.css";
import { useSnackbar } from "notistack";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { sendUploadAvatar } from "../../../api/api.ts";

export function useDebounceEffect(fn: () => void, waitTime: number, deps?: DependencyList) {
  useEffect(() => {
    const t = setTimeout(() => {
      // @ts-ignore
      fn.apply(undefined, deps);
    }, waitTime);

    return () => {
      clearTimeout(t);
    };
  }, deps);
}

const TO_RADIANS = Math.PI / 180;

export async function canvasPreview(
  image: HTMLImageElement,
  canvas: HTMLCanvasElement,
  crop: PixelCrop,
  scale = 1,
  rotate = 0,
) {
  const ctx = canvas.getContext("2d");

  if (!ctx) {
    throw new Error("No 2d context");
  }

  const scaleX = image.naturalWidth / image.width;
  const scaleY = image.naturalHeight / image.height;
  // devicePixelRatio slightly increases sharpness on retina devices
  // at the expense of slightly slower render times and needing to
  // size the image back down if you want to download/upload and be
  // true to the images natural size.
  const pixelRatio = window.devicePixelRatio;
  // const pixelRatio = 1

  canvas.width = Math.floor(crop.width * scaleX * pixelRatio);
  canvas.height = Math.floor(crop.height * scaleY * pixelRatio);

  ctx.scale(pixelRatio, pixelRatio);
  ctx.imageSmoothingQuality = "high";

  const cropX = crop.x * scaleX;
  const cropY = crop.y * scaleY;

  const rotateRads = rotate * TO_RADIANS;
  const centerX = image.naturalWidth / 2;
  const centerY = image.naturalHeight / 2;

  ctx.save();

  // 5) Move the crop origin to the canvas origin (0,0)
  ctx.translate(-cropX, -cropY);
  // 4) Move the origin to the center of the original position
  ctx.translate(centerX, centerY);
  // 3) Rotate around the origin
  ctx.rotate(rotateRads);
  // 2) Scale the image
  ctx.scale(scale, scale);
  // 1) Move the center of the image to the origin (0,0)
  ctx.translate(-centerX, -centerY);
  ctx.drawImage(image, 0, 0, image.naturalWidth, image.naturalHeight, 0, 0, image.naturalWidth, image.naturalHeight);

  ctx.restore();
}

export interface AvatarCropperDialogProps {
  open?: boolean;
  onClose: () => void;
  file?: File;
  onAvatarUpdated: () => void;
}

function centerAspectCrop(mediaWidth: number, mediaHeight: number, aspect: number) {
  return centerCrop(
    makeAspectCrop(
      {
        unit: "px",
        width: mediaWidth * 0.9,
      },
      aspect,
      mediaWidth,
      mediaHeight,
    ),
    mediaWidth,
    mediaHeight,
  );
}

const AvatarCropperDialog = ({ open, onClose, onAvatarUpdated, file }: AvatarCropperDialogProps) => {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const [src, setSrc] = useState<string | null>(null);
  const [crop, setCrop] = useState<Crop | undefined>(undefined);
  const [completedCrop, setCompletedCrop] = useState<PixelCrop>();
  const imageRef = useRef<HTMLImageElement | null>(null);
  const previewCanvasRef = useRef<HTMLCanvasElement>(null);
  const blobUrlRef = useRef("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setCrop(undefined);
      setSrc(null);
      if (file) {
        const reader = new FileReader();
        reader.addEventListener("load", () => setSrc(reader.result as string));
        reader.readAsDataURL(file);
      }
    }
  }, [open]);

  const onImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const { width, height } = e.currentTarget;
    setCrop(centerAspectCrop(width, height, 1));
  };

  const renderImage = async (): Promise<Blob> => {
    const image = imageRef.current;
    const previewCanvas = previewCanvasRef.current;
    if (!image || !previewCanvas || !completedCrop) {
      throw new Error("Crop canvas does not exist");
    }

    // This will size relative to the uploaded image
    // size. If you want to size according to what they
    // are looking at on screen, remove scaleX + scaleY
    const scaleX = image.naturalWidth / image.width;
    const scaleY = image.naturalHeight / image.height;

    const offscreen = new OffscreenCanvas(completedCrop.width * scaleX, completedCrop.height * scaleY);
    const ctx = offscreen.getContext("2d");
    if (!ctx) {
      throw new Error("No 2d context");
    }

    ctx.drawImage(
      previewCanvas,
      0,
      0,
      previewCanvas.width,
      previewCanvas.height,
      0,
      0,
      offscreen.width,
      offscreen.height,
    );
    // You might want { type: "image/jpeg", quality: <0 to 1> } to
    // reduce image size
    const imgType = "image/png";
    const blob = await offscreen.convertToBlob({
      type: imgType,
    });

    return blob;
  };

  const onSubmit = () => {
    setLoading(true);
    renderImage()
      .then((blob) => {
        dispatch(sendUploadAvatar(blob, "image/png"))
          .then(() => {
            enqueueSnackbar({
              message: t("application:setting.avatarUpdated", {}),
              variant: "success",
              action: DefaultCloseAction,
            });
            onAvatarUpdated();
            onClose();
          })
          .finally(() => {
            setLoading(false);
          });
      })
      .catch((e) => {
        enqueueSnackbar({
          message: e.message,
          variant: "error",
          action: DefaultCloseAction,
        });
        setLoading(false);
      });
  };

  useDebounceEffect(
    async () => {
      if (completedCrop?.width && completedCrop?.height && imageRef.current && previewCanvasRef.current) {
        console.log(completedCrop);
        // We use canvasPreview as it's much faster than imgPreview.
        canvasPreview(imageRef.current, previewCanvasRef.current, completedCrop, 1, 0);
      }
    },
    100,
    [completedCrop],
  );

  return (
    <DraggableDialog
      title={t("application:setting.cropAvatar", {})}
      showCancel
      showActions
      loading={loading || !src}
      disabled={!completedCrop}
      onAccept={onSubmit}
      dialogProps={{
        open: !!open,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
      }}
    >
      <DialogContent>
        {src && (
          <ReactCrop
            circularCrop
            crop={crop}
            onChange={(_, percentCrop) => setCrop(percentCrop)}
            minWidth={100}
            aspect={1}
            onComplete={(p) => setCompletedCrop(p)}
            minHeight={100}
          >
            <img
              ref={imageRef}
              onLoad={onImageLoad}
              src={src}
              alt="Avatar"
              style={{ maxWidth: "100%", maxHeight: "60vh" }}
            />
          </ReactCrop>
        )}
        {!!completedCrop && (
          <canvas
            ref={previewCanvasRef}
            style={{
              display: "none",
              border: "1px solid black",
              objectFit: "contain",
              width: completedCrop.width,
              height: completedCrop.height,
            }}
          />
        )}
      </DialogContent>
    </DraggableDialog>
  );
};

export default AvatarCropperDialog;
