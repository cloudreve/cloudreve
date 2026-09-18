import { Box, Chip, styled } from "@mui/material";
import { Add as AddIcon } from "@mui/icons-material";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { convertFileSrc } from "@tauri-apps/api/core";
import type { DriveConfig } from "./types";

const StyledChip = styled(Chip, {
  shouldForwardProp: (prop) => prop !== "selected",
})<{ selected?: boolean }>(({ theme, selected }) => ({
  marginRight: theme.spacing(0.5),
  borderWidth: 1,
  borderStyle: "solid",
  borderColor: selected ? "transparent" : theme.palette.divider,
  backgroundColor: selected ? theme.palette.action.selected : "transparent",
}));

interface DriveChipsProps {
  drives: DriveConfig[];
  selectedDrive: string | null;
  onDriveSelect: (driveId: string | null) => void;
  onAddDrive: () => void;
}

export default function DriveChips({
  drives,
  selectedDrive,
  onDriveSelect,
  onAddDrive,
}: DriveChipsProps) {
  const { t } = useTranslation();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [scrollState, setScrollState] = useState({ canScrollLeft: false, canScrollRight: false });
  const scrollTargetRef = useRef<number | null>(null);
  const scrollAnimationRef = useRef<number | null>(null);

  // Update scroll state for fade effects
  const updateScrollState = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const canScrollLeft = el.scrollLeft > 0;
    const canScrollRight = el.scrollLeft < el.scrollWidth - el.clientWidth - 1;
    setScrollState({ canScrollLeft, canScrollRight });
  }, []);

  // Handle horizontal wheel scroll with smooth animation
  const handleWheel = useCallback((e: React.WheelEvent<HTMLDivElement>) => {
    const el = scrollRef.current;
    if (!el) return;
    if (e.deltaY !== 0) {
      e.preventDefault();

      // Initialize or accumulate target scroll position
      if (scrollTargetRef.current === null) {
        scrollTargetRef.current = el.scrollLeft;
      }
      scrollTargetRef.current += e.deltaY;

      // Clamp to valid range
      const maxScroll = el.scrollWidth - el.clientWidth;
      scrollTargetRef.current = Math.max(0, Math.min(scrollTargetRef.current, maxScroll));

      // Start animation if not already running
      if (scrollAnimationRef.current === null) {
        const animate = () => {
          const target = scrollTargetRef.current;
          if (target === null || !scrollRef.current) {
            scrollAnimationRef.current = null;
            return;
          }

          const current = scrollRef.current.scrollLeft;
          const diff = target - current;

          if (Math.abs(diff) < 0.5) {
            scrollRef.current.scrollLeft = target;
            scrollTargetRef.current = null;
            scrollAnimationRef.current = null;
            updateScrollState();
            return;
          }

          // Ease toward target
          scrollRef.current.scrollLeft = current + diff * 0.15;
          updateScrollState();
          scrollAnimationRef.current = requestAnimationFrame(animate);
        };
        scrollAnimationRef.current = requestAnimationFrame(animate);
      }
    }
  }, [updateScrollState]);

  // Initialize scroll state when drives change
  useEffect(() => {
    updateScrollState();
  }, [drives, updateScrollState]);

  return (
    <Box sx={{ position: "relative", mx: -2 }}>
      {/* Left fade */}
      <Box
        sx={{
          position: "absolute",
          left: 0,
          top: 0,
          bottom: 0,
          width: 32,
          background: (theme) =>
            `linear-gradient(to right, ${theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900]}, transparent)`,
          pointerEvents: "none",
          zIndex: 1,
          opacity: scrollState.canScrollLeft ? 1 : 0,
          transition: "opacity 0.2s",
        }}
      />
      {/* Right fade */}
      <Box
        sx={{
          position: "absolute",
          right: 0,
          top: 0,
          bottom: 0,
          width: 32,
          background: (theme) =>
            `linear-gradient(to left, ${theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900]}, transparent)`,
          pointerEvents: "none",
          zIndex: 1,
          opacity: scrollState.canScrollRight ? 1 : 0,
          transition: "opacity 0.2s",
        }}
      />
      <Box
        ref={scrollRef}
        onWheel={handleWheel}
        onScroll={updateScrollState}
        sx={{
          overflowX: "auto",
          whiteSpace: "nowrap",
          pb: 0.5,
          px: 2,
          "&::-webkit-scrollbar": { display: "none" },
          scrollbarWidth: "none",
        }}
      >
        <StyledChip
          label={t("popup.allDrives", "All")}
          size="small"
          selected={selectedDrive === null}
          onClick={() => onDriveSelect(null)}
        />
        {drives.map((drive) => (
          <StyledChip
            key={drive.id}
            icon={
              drive.icon_path ? (
                <img
                  src={convertFileSrc(drive.icon_path)}
                  alt=""
                  style={{ width: 18,borderRadius:"6px", height: 18 }}
                />
              ) : undefined
            }
            label={drive.name}
            size="small"
            selected={selectedDrive === drive.id}
            onClick={() => onDriveSelect(drive.id)}
          />
        ))}
        <Chip
          icon={<AddIcon />}
          label={t("popup.newDrive", "New Drive")}
          size="small"
          color="primary"
          onClick={onAddDrive}
        />
      </Box>
    </Box>
  );
}
