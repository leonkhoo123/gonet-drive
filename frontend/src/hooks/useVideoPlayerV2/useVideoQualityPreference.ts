import { useCallback, useState } from "react";
import { isVideoQuality, type VideoQuality } from "./usePlaybackControllerCore";

/**
 * sessionStorage key remembering the quality picked in this tab/PWA session.
 *
 * sessionStorage (not localStorage) on purpose: the choice should survive
 * reloads, navigation and auto-play advances, but not outlive the tab or PWA
 * window. A fresh session starts back at Original.
 */
export const VIDEO_QUALITY_STORAGE_KEY = "videoPlayerV2:quality";

/** Read the session's remembered quality, defaulting to `original`. */
function readStoredQuality(): VideoQuality {
  try {
    const stored = sessionStorage.getItem(VIDEO_QUALITY_STORAGE_KEY);
    if (stored === null) return "original";
    // Stored as a string; numbers round-trip through Number(), "original" does not.
    const value = stored === "original" ? "original" : Number(stored);
    return isVideoQuality(value) ? value : "original";
  } catch {
    // Ignore storage failures (private mode, blocked cookies).
    return "original";
  }
}

export interface VideoQualityPreference {
  /** Quality for the clip currently open. */
  quality: VideoQuality;
  /** User pick: applies now and is remembered for the rest of the session. */
  selectQuality: (quality: VideoQuality) => void;
  /**
   * Programmatic override, e.g. the busy/failed fallback to Original. Applies
   * now but does NOT touch the stored preference, so the next clip retries the
   * quality the user actually chose.
   */
  forceQuality: (quality: VideoQuality) => void;
}

/**
 * Remembers the compress player's chosen quality for the current tab/PWA
 * session, so every clip opened afterwards starts at that quality instead of
 * back at Original.
 */
export function useVideoQualityPreference(): VideoQualityPreference {
  const [quality, setQuality] = useState<VideoQuality>(readStoredQuality);

  const selectQuality = useCallback((next: VideoQuality) => {
    setQuality(next);
    try {
      sessionStorage.setItem(VIDEO_QUALITY_STORAGE_KEY, String(next));
    } catch {
      // Ignore storage failures (private mode, quota).
    }
  }, []);

  const forceQuality = useCallback((next: VideoQuality) => {
    setQuality(next);
  }, []);

  return { quality, selectQuality, forceQuality };
}
