import { Box, Skeleton } from "@mui/material";
import { InsertDriveFile as DefaultFileIcon } from "@mui/icons-material";
import { useEffect, useMemo, useRef, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import type { FileIconResponse } from "./types";
import { rgbaToDataUrl } from "./utils";

// Global cache for file icons to avoid repeated fetches
const iconCache = new Map<string, string>();

interface FileIconProps {
  path: string;
  size?: number;
}

export default function FileIcon({ path, size = 24 }: FileIconProps) {
  const [iconUrl, setIconUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [isVisible, setIsVisible] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Get cache key based on path extension (icons are same for same file types)
  // Exception: exe files have unique icons per file, so use full path
  const cacheKey = useMemo(() => {
    const ext = path.split(".").pop()?.toLowerCase() || "unknown";
    if (ext === "exe") {
      return `${path}_${size}`;
    }
    return `${ext}_${size}`;
  }, [path, size]);

  // Intersection Observer for lazy loading
  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          setIsVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.1 }
    );

    observer.observe(element);

    return () => {
      observer.disconnect();
    };
  }, []);

  // Fetch icon only when visible
  useEffect(() => {
    if (!isVisible) return;

    let mounted = true;

    const fetchIcon = async () => {
      // Check cache first
      const cached = iconCache.get(cacheKey);
      if (cached) {
        setIconUrl(cached);
        setLoading(false);
        return;
      }

      try {
        const response = await invoke<FileIconResponse>("get_file_icon", {
          path,
          size: 64,
        });

        if (!mounted) return;

        const dataUrl = rgbaToDataUrl(
          response.data,
          response.width,
          response.height
        );

        if (dataUrl) {
          iconCache.set(cacheKey, dataUrl);
          setIconUrl(dataUrl);
        } else {
          setError(true);
        }
      } catch (err) {
        console.error("Failed to fetch file icon:", err);
        if (mounted) {
          setError(true);
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };

    fetchIcon();

    return () => {
      mounted = false;
    };
  }, [isVisible, path, size, cacheKey]);

  // Show placeholder until visible
  if (!isVisible) {
    return (
      <Box
        ref={containerRef}
        sx={{ width: size, height: size, display: "flex", alignItems: "center", justifyContent: "center" }}
      >
        <DefaultFileIcon sx={{ fontSize: size }} color="action" />
      </Box>
    );
  }

  if (loading) {
    return <Skeleton variant="rectangular" width={size} height={size} />;
  }

  if (error || !iconUrl) {
    return <DefaultFileIcon sx={{ fontSize: size }} color="action" />;
  }

  return (
    <Box
      component="img"
      src={iconUrl}
      alt="file icon"
      sx={{
        width: size,
        height: size,
        objectFit: "contain",
      }}
    />
  );
}
