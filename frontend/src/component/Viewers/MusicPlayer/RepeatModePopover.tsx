import { Box, Divider, Popover, ToggleButton, ToggleButtonGroup, Typography, styled } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import ArrowRepeatAll from "../../Icons/ArrowRepeatAll.tsx";
import ArrowRepeatOne from "../../Icons/ArrowRepeatOne.tsx";
import ArrowRepeatOff from "../../Icons/ArrowRepeatOff.tsx";
import ArrowShuffle from "../../Icons/ArrowShuffle.tsx";
import { LoopMode } from "./MusicPlayer.tsx";

interface RepeatModePopoverProps {
  open?: boolean;
  anchorEl?: HTMLElement | null;
  onClose?: () => void;
  loopMode: number;
  onLoopModeChange: (mode: number) => void;
  playbackSpeed: number;
  onPlaybackSpeedChange: (speed: number) => void;
}

const NoWrapToggleButton = styled(ToggleButton)({
  whiteSpace: "nowrap",
});

export const RepeatModePopover = ({
  open,
  anchorEl,
  onClose,
  loopMode,
  onLoopModeChange,
  playbackSpeed,
  onPlaybackSpeedChange,
}: RepeatModePopoverProps) => {
  const { t } = useTranslation();

  const currentLoopMode = useMemo(() => {
    switch (loopMode) {
      case LoopMode.list_repeat:
        return "list_repeat";
      case LoopMode.single_repeat:
        return "single_repeat";
      case LoopMode.shuffle:
        return "shuffle";
      case LoopMode.play_once:
        return "play_once";
      default:
        return "list_repeat";
    }
  }, [loopMode]);

  const currentSpeed = useMemo(() => {
    return playbackSpeed.toString();
  }, [playbackSpeed]);

  const handleLoopModeChange = (_event: React.MouseEvent<HTMLElement>, newMode: string) => {
    if (!newMode) return;

    let newLoopMode: number;
    switch (newMode) {
      case "list_repeat":
        newLoopMode = LoopMode.list_repeat;
        break;
      case "single_repeat":
        newLoopMode = LoopMode.single_repeat;
        break;
      case "shuffle":
        newLoopMode = LoopMode.shuffle;
        break;
      case "play_once":
        newLoopMode = LoopMode.play_once;
        break;
      default:
        return;
    }
    onLoopModeChange(newLoopMode);
  };

  const handleSpeedChange = (_event: React.MouseEvent<HTMLElement>, newSpeed: string) => {
    if (!newSpeed) return;
    const speed = parseFloat(newSpeed);
    if (!isNaN(speed)) {
      onPlaybackSpeedChange(speed);
    }
  };

  return (
    <Popover
      open={!!open}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "center",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "center",
      }}
    >
      <Box sx={{ p: 2, minWidth: 300 }}>
        <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
          {t("fileManager.repeatMode")}
        </Typography>
        <ToggleButtonGroup
          color="primary"
          value={currentLoopMode}
          exclusive
          onChange={handleLoopModeChange}
          size="small"
          fullWidth
          sx={{ mb: 2 }}
        >
          <NoWrapToggleButton value="list_repeat">
            <ArrowRepeatAll fontSize="small" sx={{ mr: 1 }} />
            {t("fileManager.listRepeat")}
          </NoWrapToggleButton>
          <NoWrapToggleButton value="single_repeat">
            <ArrowRepeatOne fontSize="small" sx={{ mr: 1 }} />
            {t("fileManager.singleRepeat")}
          </NoWrapToggleButton>
          <NoWrapToggleButton value="shuffle">
            <ArrowShuffle fontSize="small" sx={{ mr: 1 }} />
            {t("fileManager.shuffle")}
          </NoWrapToggleButton>
          <NoWrapToggleButton value="play_once">
            <ArrowRepeatOff fontSize="small" sx={{ mr: 1 }} />
            {t("fileManager.playOnce")}
          </NoWrapToggleButton>
        </ToggleButtonGroup>

        <Divider sx={{ mb: 2 }} />

        <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
          {t("fileManager.playbackSpeed")}
        </Typography>
        <ToggleButtonGroup
          color="primary"
          value={currentSpeed}
          exclusive
          onChange={handleSpeedChange}
          size="small"
          fullWidth
        >
          <ToggleButton value="0.5">0.5×</ToggleButton>
          <ToggleButton value="0.75">0.75×</ToggleButton>
          <ToggleButton value="1">1×</ToggleButton>
          <ToggleButton value="1.25">1.25×</ToggleButton>
          <ToggleButton value="1.5">1.5×</ToggleButton>
          <ToggleButton value="2">2×</ToggleButton>
        </ToggleButtonGroup>
      </Box>
    </Popover>
  );
};

export default RepeatModePopover;
