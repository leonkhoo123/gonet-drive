import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type RefObject,
} from "react";

/**
 * Owns the native <video> element playback state (playing/time/duration/buffer)
 * and the core transport actions. All mutations go through the shared videoRef
 * so the surface and controls stay in sync.
 */
export function useVideoPlayback(
  videoRef: RefObject<HTMLVideoElement | null>
) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1.0);
  const [progress, setProgress] = useState(0);
  const [bufferedProgress, setBufferedProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const wasPlayingBeforeScrub = useRef(false);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => {
      setIsPlaying(true);
    };
    const onPause = () => {
      setIsPlaying(false);
    };
    const onLoaded = () => {
      setDuration(video.duration || 0);
    };
    const onTime = () => {
      setCurrentTime(video.currentTime);
      setProgress((video.currentTime / video.duration) * 100);
    };
    const onProgress = () => {
      if (!video.duration || video.buffered.length === 0) return;
      const end = video.buffered.end(video.buffered.length - 1);
      setBufferedProgress((end / video.duration) * 100);
    };
    // Reaching the end pauses the element, but `pause` is not guaranteed to
    // fire, so mirror the playing state here explicitly.
    const onEnded = () => {
      setIsPlaying(false);
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("loadedmetadata", onLoaded);
    video.addEventListener("timeupdate", onTime);
    video.addEventListener("progress", onProgress);
    video.addEventListener("ended", onEnded);

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("loadedmetadata", onLoaded);
      video.removeEventListener("timeupdate", onTime);
      video.removeEventListener("progress", onProgress);
      video.removeEventListener("ended", onEnded);
    };
  }, [videoRef]);

  /** Play, swallowing autoplay rejections the same way the original did. */
  const safePlay = useCallback(() => {
    const v = videoRef.current;
    if (!v) return;
    v.play().catch(() => {
      setIsPlaying(false);
    });
  }, [videoRef]);

  const togglePlay = useCallback(() => {
    const v = videoRef.current;
    if (!v) return;
    if (isPlaying) {
      v.pause();
    } else {
      void v.play();
    }
  }, [isPlaying, videoRef]);

  const skip = useCallback(
    (sec: number) => {
      const v = videoRef.current;
      if (v) v.currentTime += sec;
    },
    [videoRef]
  );

  const changeSpeed = (rate: number) => {
    const v = videoRef.current;
    if (!v) return;
    v.playbackRate = rate;
    setPlaybackRate(rate);
  };

  /** Seek to an absolute time and mirror it into the UI state. */
  const seekTo = useCallback(
    (newTime: number) => {
      const v = videoRef.current;
      if (!v) return;
      v.currentTime = newTime;
      setCurrentTime(newTime);
      if (duration > 0) {
        setProgress((newTime / duration) * 100);
      }
    },
    [duration, videoRef]
  );

  /** Pause (remembering whether it was playing) before a scrub begins. */
  const scrubStart = useCallback(() => {
    const v = videoRef.current;
    if (v) {
      wasPlayingBeforeScrub.current = !v.paused;
      v.pause();
    }
  }, [videoRef]);

  /** Resume playback after a scrub if it was playing beforehand. */
  const scrubEnd = useCallback(() => {
    if (wasPlayingBeforeScrub.current) {
      safePlay();
    }
  }, [safePlay]);

  /** Handle a percent value coming from the seek bar. */
  const scrubTo = useCallback(
    (newProgress: number) => {
      const v = videoRef.current;
      if (!v || !duration) return;

      const newTime = (newProgress / 100) * duration;

      v.currentTime = newTime;

      setProgress(newProgress);
      setCurrentTime(newTime);
    },
    [duration, videoRef]
  );

  return {
    isPlaying,
    playbackRate,
    progress,
    bufferedProgress,
    duration,
    currentTime,
    wasPlayingBeforeScrub,
    safePlay,
    togglePlay,
    skip,
    changeSpeed,
    seekTo,
    scrubStart,
    scrubEnd,
    scrubTo,
  };
}
