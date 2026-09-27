import type { RefObject } from "react";
import {
  usePlaybackControllerCore,
  type PlaybackControllerState,
  type VideoQuality,
} from "./usePlaybackControllerCore";

/**
 * Native compressed delivery: the transcode streams straight into
 * `<video src>` (Chromium/Firefox), which tolerates the chunked, non-seekable
 * fragmented MP4.
 */
export function useNativePlaybackController(
  videoRef: RefObject<HTMLVideoElement | null>,
  fileUrl: string,
  filePath: string,
  isOpen: boolean,
  quality: VideoQuality
): PlaybackControllerState {
  return usePlaybackControllerCore(
    videoRef,
    fileUrl,
    filePath,
    isOpen,
    quality,
    "native"
  );
}
