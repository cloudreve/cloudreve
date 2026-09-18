import { grey } from "@mui/material/colors";
import { heicTo, isHeic } from "heic-to";
import * as LivePhotosKit from "livephotoskit";
import React, { useEffect } from "react";
import { getFileEntityUrl, getFileInfo } from "../../../../api/api.ts";
import { EntityType, FileResponse, Metadata } from "../../../../api/explorer.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { getFileLinkedUri } from "../../../../util";
import { LRUCache } from "../../../../util/lru.ts";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import useMountedRef from "./hooks/useMountedRef";
import "./Photo.less";
import type { BrokenElementParams } from "./types";

export interface IPhotoLoadedParams {
  loaded?: boolean;
  naturalWidth?: number;
  naturalHeight?: number;
  broken?: boolean;
}

export interface IPhotoProps extends React.HTMLAttributes<HTMLElement> {
  file: FileResponse;
  version?: string;
  loaded: boolean;
  broken: boolean;
  onPhotoLoad: (params: IPhotoLoadedParams) => void;
  loadingElement?: JSX.Element;
  brokenElement?: JSX.Element | ((photoProps: BrokenElementParams) => JSX.Element);
}

// Global LRU cache for HEIC conversions (capacity: 50 images)
const heicConversionCache = new LRUCache<string, string>(50);

export default function Photo({
  file,
  version,
  loaded,
  broken,
  className,
  onPhotoLoad,
  loadingElement,
  brokenElement,
  ...restProps
}: IPhotoProps) {
  const mountedRef = useMountedRef();
  const dispatch = useAppDispatch();
  const [imageSrc, setImageSrc] = React.useState<string | undefined>(undefined);
  const playerRef = React.useRef<LivePhotosKit.Player | null>(null);

  // Helper function to check if file is HEIC/HEIF based on extension
  const isHeicFile = (fileName: string): boolean => {
    const extension = fileName.toLowerCase().split(".").pop();
    return extension === "heic" || extension === "heif";
  };

  // Helper function to convert HEIC to JPG with caching
  const convertHeicToJpg = async (imageUrl: string, cacheKey: string): Promise<string> => {
    // Check cache first
    const cachedUrl = heicConversionCache.get(cacheKey);
    if (cachedUrl) {
      return cachedUrl;
    }

    try {
      // Fetch the image as blob
      const response = await fetch(imageUrl);
      const blob = await response.blob();

      // Convert blob to File object for isHeic check
      const file = new File([blob], "image", { type: blob.type });

      // Check if it's actually a HEIC file
      const isHeicBlob = await isHeic(file);

      if (isHeicBlob) {
        // Convert HEIC to JPG
        const jpgBlob = await heicTo({
          blob: blob,
          type: "image/jpeg",
          quality: 1,
        });

        // Create object URL for the converted image
        const convertedUrl = URL.createObjectURL(jpgBlob);

        // Cache the converted URL
        heicConversionCache.set(cacheKey, convertedUrl);

        return convertedUrl;
      } else {
        // If not HEIC, cache and return original URL
        heicConversionCache.set(cacheKey, imageUrl);
        return imageUrl;
      }
    } catch (error) {
      console.error("Error converting HEIC to JPG:", error);
      throw error;
    }
  };

  useEffect(() => {
    dispatch(
      getFileEntityUrl({
        uris: [getFileLinkedUri(file)],
        entity: version,
      }),
    )
      .then(async (res) => {
        const originalUrl = res.urls[0].url;
        const cacheKey = `${file.id}-${version || "default"}`;

        // Check if the file is HEIC/HEIF and convert if needed
        if (isHeicFile(file.name)) {
          try {
            const convertedUrl = await convertHeicToJpg(originalUrl, cacheKey);
            setImageSrc(convertedUrl);
            if (file.metadata?.[Metadata.live_photo]) {
              loadLivePhoto(file, convertedUrl);
            }
          } catch (error) {
            console.error("Failed to convert HEIC image:", error);
            if (mountedRef.current) {
              onPhotoLoad({
                broken: true,
              });
            }
          }
        } else {
          setImageSrc(originalUrl);
          loadLivePhoto(file, originalUrl);
        }
      })
      .catch((e) => {
        if (mountedRef.current) {
          onPhotoLoad({
            broken: true,
          });
        }
      });
  }, []);

  const loadLivePhoto = async (file: FileResponse, imgUrl: string) => {
    if (!file.metadata?.[Metadata.live_photo]) {
      return;
    }

    try {
      const fileExtended = await dispatch(
        getFileInfo(
          {
            uri: getFileLinkedUri(file),
            extended: true,
          },
          true,
        ),
      );

      // find live photo entity
      const livePhotoEntity = fileExtended?.extended_info?.entities?.find(
        (entity) => entity.type === EntityType.live_photo,
      );

      // get live photo entity url
      const livePhotoEntityUrl = await dispatch(
        getFileEntityUrl({
          uris: [getFileLinkedUri(file)],
          entity: livePhotoEntity?.id,
        }),
      );

      const imgElement = document.getElementById(file.id);
      if (imgElement) {
        const player = LivePhotosKit.Player(imgElement as HTMLElement);
        playerRef.current = player;
        player.photoSrc = imgUrl;
        player.videoSrc = livePhotoEntityUrl.urls[0].url;
        player.proactivelyLoadsVideo = true;
      }
    } catch (e) {
      console.error("Failed to load live photo:", e);
    }
  };

  function handleImageLoaded(e: React.SyntheticEvent<HTMLImageElement>) {
    const { naturalWidth, naturalHeight } = e.target as HTMLImageElement;
    if (mountedRef.current) {
      onPhotoLoad({
        loaded: true,
        naturalWidth,
        naturalHeight,
      });
    }
  }

  function handleImageBroken(e: React.SyntheticEvent<HTMLImageElement>) {
    console.log("handleImageBroken", e);
    if (mountedRef.current) {
      onPhotoLoad({
        broken: true,
      });
    }
  }

  // Clean up object URL when component unmounts or imageSrc changes
  useEffect(() => {
    return () => {
      if (imageSrc && imageSrc.startsWith("blob:")) {
        // Don't revoke cached URLs, let the cache handle cleanup
        // URL.revokeObjectURL(imageSrc);
      }
    };
  }, [imageSrc]);

  const { onMouseDown, onTouchStart, style, ...rest } = restProps;

  // Extract width and height from style if available
  const { width, height, ...restStyle } = style || {};

  useEffect(() => {
    if (playerRef.current) {
      // Convert width and height to numbers, defaulting to 0 if not valid
      const numWidth = typeof width === "number" ? width : 0;
      const numHeight = typeof height === "number" ? height : 0;
      playerRef.current.updateSize(numWidth, numHeight);
    }
  }, [width, height]);

  if (file && !broken) {
    return (
      <>
        {imageSrc && (
          <div onMouseDown={onMouseDown} onTouchStart={onTouchStart} style={{ width, height }}>
            <img
              id={file?.id ?? "photobox-img"}
              className={`PhotoView__Photo${className ? ` ${className}` : ""}`}
              src={imageSrc}
              draggable={false}
              onLoad={handleImageLoaded}
              onError={handleImageBroken}
              alt=""
              style={{ width, height, ...restStyle }}
              {...rest}
            />
          </div>
        )}
        {!loaded && (
          <FacebookCircularProgress
            sx={{ position: "relative", top: "-20px", left: "-20px" }}
            fgColor={"#fff"}
            bgColor={grey[800]}
          />
        )}
      </>
    );
  }

  if (brokenElement) {
    return (
      <span className="PhotoView__icon">
        {typeof brokenElement === "function" ? brokenElement({ src: imageSrc ?? "" }) : brokenElement}
      </span>
    );
  }

  return null;
}
