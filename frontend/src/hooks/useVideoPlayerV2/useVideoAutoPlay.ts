import { useCallback, useEffect, useRef, useState } from "react";
import type { RefObject } from "react";
import type { FileInterface } from "@/api/api-file";
import {
  getUnplayedVideos,
  markVideoPlayed,
  resetPlayedVideos,
} from "@/utils/videoShuffleStore";

/** localStorage key remembering the selected play mode. */
const AUTOPLAY_STORAGE_KEY = "videoPlayerV2:autoPlay";

/**
 * What the player does when the current clip ends:
 * - `off`     — stay on the finished clip
 * - `auto`    — open the next clip in the folder's list order
 * - `shuffle` — open a random unplayed clip until every clip has played once
 */
export type AutoPlayMode = "off" | "auto" | "shuffle";

/** Cycle order used by the single top-right toggle button. */
const MODE_CYCLE: AutoPlayMode[] = ["off", "auto", "shuffle"];

/** Read the persisted mode, migrating the previous boolean representation. */
function readStoredMode(): AutoPlayMode {
  try {
    const stored = localStorage.getItem(AUTOPLAY_STORAGE_KEY);
    if (stored === "off" || stored === "auto" || stored === "shuffle") {
      return stored;
    }
    if (stored === "true") return "auto";
    return "off";
  } catch {
    return "off";
  }
}

/** Folder that owns a clip, so shuffle keeps one bag per folder. */
function folderOf(filePath: string): string {
  const slash = filePath.lastIndexOf("/");
  return slash > 0 ? filePath.slice(0, slash) : "";
}

interface UseVideoAutoPlayParams {
  isOpen: boolean;
  videoRef: RefObject<HTMLVideoElement | null>;
  /** Path of the clip currently open. */
  filePath: string;
  /** Video files of the current folder, in the list's current sort order. */
  videoFiles: FileInterface[];
  /** Swap the open clip; the parent updates its selection. */
  onSelectVideo: (file: FileInterface) => void;
  /**
   * True while the open clip has marks (rename / disqualified / rotation)
   * that were not saved yet. Advancing is held so nothing is silently lost.
   */
  holdAdvance: boolean;
}

/**
 * Drives what happens when a clip ends. The mode is chosen with a single
 * cycling button and persisted in localStorage (defaults to off). In `auto`
 * the next clip in list order is opened; in `shuffle` a random not-yet-played
 * clip is opened, stopping once the folder's session bag is exhausted. Both
 * modes hold when the finished clip has unsaved marks.
 */
export function useVideoAutoPlay({
  isOpen,
  videoRef,
  filePath,
  videoFiles,
  onSelectVideo,
  holdAdvance,
}: UseVideoAutoPlayParams) {
  const [mode, setMode] = useState<AutoPlayMode>(readStoredMode);

  const cycleMode = useCallback(() => {
    setMode((prev) => {
      const next =
        MODE_CYCLE[(MODE_CYCLE.indexOf(prev) + 1) % MODE_CYCLE.length];
      try {
        localStorage.setItem(AUTOPLAY_STORAGE_KEY, next);
      } catch {
        // Ignore storage failures (private mode, quota).
      }
      return next;
    });
  }, []);

  const folderPath = folderOf(filePath);

  // Keep the latest values in refs so the ended listener stays stable and
  // never reads a stale snapshot.
  const modeRef = useRef(mode);
  modeRef.current = mode;
  const filePathRef = useRef(filePath);
  filePathRef.current = filePath;
  const folderPathRef = useRef(folderPath);
  folderPathRef.current = folderPath;
  const videoFilesRef = useRef(videoFiles);
  videoFilesRef.current = videoFiles;
  const holdAdvanceRef = useRef(holdAdvance);
  holdAdvanceRef.current = holdAdvance;
  const onSelectRef = useRef(onSelectVideo);
  onSelectRef.current = onSelectVideo;

  const prevModeRef = useRef<AutoPlayMode>("off");

  /* Track played clips while shuffle is active. Entering shuffle with an
     already-exhausted bag starts a fresh cycle so re-engaging it works. */
  useEffect(() => {
    if (!isOpen || mode !== "shuffle") {
      prevModeRef.current = mode;
      return;
    }
    const enteringShuffle = prevModeRef.current !== "shuffle";
    prevModeRef.current = mode;

    if (
      enteringShuffle &&
      getUnplayedVideos(folderPath, videoFilesRef.current).length === 0
    ) {
      resetPlayedVideos(folderPath);
    }
    if (filePath) markVideoPlayed(folderPath, filePath);
  }, [isOpen, mode, folderPath, filePath]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !isOpen || mode === "off") return;

    const handleEnded = () => {
      if (holdAdvanceRef.current) return;
      const currentMode = modeRef.current;
      const files = videoFilesRef.current;

      if (currentMode === "auto") {
        const index = files.findIndex((f) => f.path === filePathRef.current);
        const next = index >= 0 ? files[index + 1] : undefined;
        if (next) onSelectRef.current(next);
        return;
      }

      if (currentMode === "shuffle") {
        const remaining = getUnplayedVideos(folderPathRef.current, files);
        if (remaining.length === 0) return;
        const next =
          remaining[Math.floor(Math.random() * remaining.length)];
        onSelectRef.current(next);
      }
    };

    video.addEventListener("ended", handleEnded);
    return () => {
      video.removeEventListener("ended", handleEnded);
    };
  }, [isOpen, mode, videoRef]);

  /* Derived info for the toggle button. */
  const remaining = getUnplayedVideos(folderPath, videoFiles).filter(
    (file) => file.path !== filePath
  ).length;

  const currentIndex = videoFiles.findIndex((f) => f.path === filePath);
  const hasNext =
    mode === "auto"
      ? currentIndex >= 0 && currentIndex + 1 < videoFiles.length
      : mode === "shuffle"
        ? remaining > 0
        : false;

  return { mode, cycleMode, hasNext, shuffleRemaining: remaining };
}
