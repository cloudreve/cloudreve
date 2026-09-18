import { IconButton, Tooltip } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
import { getFileEntityUrl } from "../../../api/api.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import SessionManager, { UserSettings } from "../../../session";
import { getFileLinkedUri } from "../../../util";
import MusicNote2 from "../../Icons/MusicNote2.tsx";
import MusicNote2Play from "../../Icons/MusicNote2Play.tsx";
import PlayerPopup from "./PlayerPopup.tsx";

export const LoopMode = {
  list_repeat: 0,
  single_repeat: 1,
  shuffle: 2,
  play_once: 3,
};

const MusicPlayer = () => {
  const dispatch = useAppDispatch();
  const playerState = useAppSelector((state) => state.globalState.musicPlayer);
  const audio = useRef<HTMLAudioElement>(null);
  const icon = useRef<HTMLButtonElement>(null);
  const [playing, setPlaying] = useState(false);
  const [volume, setVolume] = useState(0.2);
  const [index, setIndex] = useState<number | undefined>(undefined);
  const [loading, setLoading] = useState(false);
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [duration, setDuration] = useState(0);
  const [current, setCurrent] = useState(0);
  const [loopMode, setLoopMode] = useState(LoopMode.list_repeat);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const playHistory = useRef<number[]>([]);

  useEffect(() => {
    if (playerState) {
      playHistory.current = [];
      setPlaying(true);
      setPopoverOpen(true);
      const volume = SessionManager.getWithFallback(UserSettings.MusicVolume);
      setVolume(volume);
      playIndex(playerState.startIndex, volume);
    }

    audio.current?.addEventListener("timeupdate", timeUpdate);
    return () => {
      setPlaying(false);
      audio.current?.removeEventListener("timeupdate", timeUpdate);
    };
  }, [playerState]);

  const playIndex = useCallback(
    async (index: number, latestVolume?: number) => {
      if (audio.current && playerState) {
        audio.current.pause();
        setIndex(index);
        try {
          const res = await dispatch(
            getFileEntityUrl({
              uris: [getFileLinkedUri(playerState.files[index])],
              entity: playerState.version,
            }),
          );
          audio.current.src = res.urls[0].url;
          audio.current.currentTime = 0;
          audio.current.play();
          audio.current.volume = latestVolume ?? volume;
          audio.current.playbackRate = playbackSpeed;
        } catch (e) {
          console.error(e);
        }
      }
    },
    [playerState, volume, playbackSpeed],
  );

  const loopProceed = useCallback(
    (isNext: boolean) => {
      if (!playerState) {
        return;
      }

      switch (loopMode) {
        case LoopMode.list_repeat:
        case LoopMode.play_once:
          if (isNext) {
            playIndex(((index ?? 0) + 1) % playerState?.files.length);
          } else {
            playIndex(((index ?? 0) - 1 + playerState?.files.length) % playerState?.files.length);
          }
          break;
        case LoopMode.single_repeat:
          playIndex(index ?? 0);
          break;
        case LoopMode.shuffle:
          if (isNext) {
            playHistory.current.push(index ?? 0);
            const nextIndex = Math.floor(Math.random() * playerState?.files.length);
            playIndex(nextIndex);
          } else {
            playHistory.current.pop();
            playIndex(playHistory.current.pop() ?? index ?? 0);
          }
          break;
      }
    },
    [loopMode, playIndex, playerState, index],
  );

  const manualNavigate = useCallback(
    (isNext: boolean) => {
      if (!playerState) {
        return;
      }

      if (loopMode === LoopMode.single_repeat) {
        if (isNext) {
          playIndex(((index ?? 0) + 1) % playerState?.files.length);
        } else {
          playIndex(((index ?? 0) - 1 + playerState?.files.length) % playerState?.files.length);
        }
      } else {
        loopProceed(isNext);
      }
    },
    [loopMode, playIndex, playerState, index, loopProceed],
  );

  const onPlayEnded = useCallback(() => {
    if (loopMode !== LoopMode.play_once) {
      loopProceed(true);
    }
  }, [loopMode, loopProceed]);

  const timeUpdate = useCallback(() => {
    setCurrent(Math.floor(audio.current?.currentTime || 0));
    setDuration(Math.floor(audio.current?.duration || 0));
  }, []);

  const seek = useCallback((time: number) => {
    if (audio.current) {
      audio.current.currentTime = time;
    }
  }, []);

  const playingTooltip = playerState
    ? `[${(index ?? 0) + 1}/${playerState.files.length}] ${playerState?.files[index ?? 0]?.name}`
    : "";

  const onPlayerPopoverClose = useCallback(() => {
    setPopoverOpen(false);
  }, []);

  const onPlayerPopoverOpen = useCallback(() => {
    setPopoverOpen(true);
  }, []);

  const togglePause = useCallback(() => {
    if (audio.current) {
      if (audio.current.paused) {
        audio.current.play();
        setPlaying(true);
      } else {
        audio.current.pause();
        setPlaying(false);
      }
    }
  }, []);

  const setVolumeLevel = useCallback((volume: number) => {
    if (audio.current) {
      audio.current.volume = volume;
      setVolume(volume);
    }
  }, []);

  const toggleLoopMode = useCallback(() => {
    setLoopMode((loopMode) => (loopMode + 1) % 4);
  }, []);

  const setLoopModeHandler = useCallback((mode: number) => {
    setLoopMode(mode);
  }, []);

  const setPlaybackSpeedHandler = useCallback((speed: number) => {
    setPlaybackSpeed(speed);
    if (audio.current) {
      audio.current.playbackRate = speed;
    }
  }, []);

  return (
    <>
      <audio ref={audio} onPause={() => setPlaying(false)} onPlay={() => setPlaying(true)} onEnded={onPlayEnded} />
      <Tooltip title={playingTooltip} enterDelay={0}>
        <IconButton ref={icon} onClick={onPlayerPopoverOpen} size="large">
          {playing ? <MusicNote2Play /> : <MusicNote2 />}
        </IconButton>
      </Tooltip>
      {index !== undefined && (
        <PlayerPopup
          playIndex={playIndex}
          loopProceed={loopProceed}
          manualNavigate={manualNavigate}
          file={playerState?.files[index]}
          duration={duration}
          current={current}
          open={popoverOpen}
          setVolumeLevel={setVolumeLevel}
          volume={volume}
          onSeek={seek}
          togglePause={togglePause}
          playing={playing}
          playlist={playerState?.files}
          loopMode={loopMode}
          toggleLoopMode={toggleLoopMode}
          setLoopMode={setLoopModeHandler}
          playbackSpeed={playbackSpeed}
          setPlaybackSpeed={setPlaybackSpeedHandler}
          anchorEl={icon.current}
          onClose={onPlayerPopoverClose}
        />
      )}
    </>
  );
};

export default MusicPlayer;
