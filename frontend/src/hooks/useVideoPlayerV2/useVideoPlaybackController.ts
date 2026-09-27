import type { RefObject } from "react";
import { shouldUseMse } from "./mseStream";
import { useNativePlaybackController } from "./useNativePlaybackController";
import { useVideoPlaybackControllerIOS } from "./useVideoPlaybackControllerIOS";
import type {
  PlaybackControllerState,
  VideoQuality,
} from "./usePlaybackControllerCore";

export type { PlaybackControllerState, VideoQuality };

/**
 * Platform-aware playback controller used by the compress player.
 *
 * The transport is chosen once, at module load: WebKit on Apple platforms
 * cannot play the chunked, non-seekable transcode natively, so it uses the
 * MSE controller; every other browser keeps the native streaming path. The
 * choice is constant for the app's lifetime, so calling the selected hook
 * unconditionally is safe.
 */
const usePlatformPlaybackController = shouldUseMse()
  ? useVideoPlaybackControllerIOS
  : useNativePlaybackController;

export function useVideoPlaybackController(
  videoRef: RefObject<HTMLVideoElement | null>,
  fileUrl: string,
  filePath: string,
  isOpen: boolean,
  quality: VideoQuality
): PlaybackControllerState {
  return usePlatformPlaybackController(
    videoRef,
    fileUrl,
    filePath,
    isOpen,
    quality
  );
}
