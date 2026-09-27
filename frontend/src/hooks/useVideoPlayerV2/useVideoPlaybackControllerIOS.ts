import type { RefObject } from "react";
import {
  usePlaybackControllerCore,
  type PlaybackControllerState,
  type VideoQuality,
} from "./usePlaybackControllerCore";

/**
 * iOS/iPadOS (and desktop Safari) controller: identical timeline and actions to
 * the native player, but compressed segments are fetched and fed into a
 * SourceBuffer because WebKit refuses the chunked, non-seekable fMP4.
 */
export function useVideoPlaybackControllerIOS(
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
    "mse"
  );
}
