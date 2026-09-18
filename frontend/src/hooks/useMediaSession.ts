import { useCallback, useEffect } from "react";
import { FileResponse, Metadata } from "../api/explorer";

interface MediaSessionConfig {
  file?: FileResponse;
  playing: boolean;
  duration: number;
  current: number;
  thumbSrc?: string | null;
  onPlay: () => void;
  onPause: () => void;
  onPrevious: () => void;
  onNext: () => void;
  onSeek?: (time: number) => void;
}

export const useMediaSession = ({
  file,
  playing,
  duration,
  current,
  thumbSrc,
  onPlay,
  onPause,
  onPrevious,
  onNext,
  onSeek,
}: MediaSessionConfig) => {
  // Update media session metadata
  const updateMetadata = useCallback(() => {
    if (!("mediaSession" in navigator) || !file) {
      return;
    }

    const title = file.metadata?.[Metadata.music_title] ?? file.name;
    const artist = file.metadata?.[Metadata.music_artist] ?? "";
    const album = file.metadata?.[Metadata.music_album] ?? "";

    // Prepare artwork array
    const artwork: MediaImage[] = [];
    if (thumbSrc) {
      // Add multiple sizes for better compatibility
      artwork.push(
        { src: thumbSrc, sizes: "96x96", type: "image/jpeg" },
        { src: thumbSrc, sizes: "128x128", type: "image/jpeg" },
        { src: thumbSrc, sizes: "192x192", type: "image/jpeg" },
        { src: thumbSrc, sizes: "256x256", type: "image/jpeg" },
        { src: thumbSrc, sizes: "384x384", type: "image/jpeg" },
        { src: thumbSrc, sizes: "512x512", type: "image/jpeg" },
      );
    }

    navigator.mediaSession.metadata = new MediaMetadata({
      title,
      artist,
      album,
      artwork,
    });
  }, [file, thumbSrc]);

  // Update playback state
  const updatePlaybackState = useCallback(() => {
    if (!("mediaSession" in navigator)) {
      return;
    }

    navigator.mediaSession.playbackState = playing ? "playing" : "paused";
  }, [playing]);

  // Update position state
  const updatePositionState = useCallback(() => {
    if (!("mediaSession" in navigator) || !duration || duration <= 0) {
      return;
    }

    try {
      navigator.mediaSession.setPositionState({
        duration,
        playbackRate: 1,
        position: current,
      });
    } catch (error) {
      // Some browsers may not support position state
      console.debug("Media Session position state not supported:", error);
    }
  }, [duration, current]);

  // Set up action handlers
  const setupActionHandlers = useCallback(() => {
    if (!("mediaSession" in navigator)) {
      return;
    }

    // Play action
    navigator.mediaSession.setActionHandler("play", () => {
      onPlay();
    });

    // Pause action
    navigator.mediaSession.setActionHandler("pause", () => {
      onPause();
    });

    // Previous track action
    navigator.mediaSession.setActionHandler("previoustrack", () => {
      onPrevious();
    });

    // Next track action
    navigator.mediaSession.setActionHandler("nexttrack", () => {
      onNext();
    });

    // Seek backward action
    navigator.mediaSession.setActionHandler("seekbackward", (details) => {
      if (onSeek) {
        const seekTime = Math.max(0, current - (details.seekOffset || 10));
        onSeek(seekTime);
      }
    });

    // Seek forward action
    navigator.mediaSession.setActionHandler("seekforward", (details) => {
      if (onSeek) {
        const seekTime = Math.min(duration, current + (details.seekOffset || 10));
        onSeek(seekTime);
      }
    });

    // Seek to action (for scrubbing)
    navigator.mediaSession.setActionHandler("seekto", (details) => {
      if (onSeek && details.seekTime !== undefined) {
        onSeek(details.seekTime);
      }
    });

    // Stop action
    navigator.mediaSession.setActionHandler("stop", () => {
      onPause();
    });
  }, [onPlay, onPause, onPrevious, onNext, onSeek, current, duration]);

  // Clean up action handlers
  const cleanupActionHandlers = useCallback(() => {
    if (!("mediaSession" in navigator)) {
      return;
    }

    const actions: MediaSessionAction[] = [
      "play",
      "pause",
      "previoustrack",
      "nexttrack",
      "seekbackward",
      "seekforward",
      "seekto",
      "stop",
    ];

    actions.forEach((action) => {
      try {
        navigator.mediaSession.setActionHandler(action, null);
      } catch (error) {
        // Some browsers may not support all actions
        console.debug(`Media Session action ${action} not supported:`, error);
      }
    });
  }, []);

  // Initialize media session when component mounts
  useEffect(() => {
    setupActionHandlers();

    // Cleanup on unmount
    return () => {
      cleanupActionHandlers();
    };
  }, [setupActionHandlers, cleanupActionHandlers]);

  // Update metadata when file or thumbnail changes
  useEffect(() => {
    updateMetadata();
  }, [updateMetadata]);

  // Update playback state when playing status changes
  useEffect(() => {
    updatePlaybackState();
  }, [updatePlaybackState]);

  // Update position state when duration or current position changes
  useEffect(() => {
    updatePositionState();
  }, [updatePositionState]);

  // Return utility functions for manual control if needed
  return {
    updateMetadata,
    updatePlaybackState,
    updatePositionState,
    setupActionHandlers,
    cleanupActionHandlers,
  };
};
