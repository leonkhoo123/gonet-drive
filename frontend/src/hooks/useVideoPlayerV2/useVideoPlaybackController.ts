import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { getVideoDuration } from "@/api/api-video";
import { buildCompressStreamUrl } from "@/utils/videoCompress";

/**
 * Selectable playback qualities.
 *
 * - `"original"` streams the untouched file through the normal Range endpoint
 *   and seeks natively (`video.currentTime`).
 * - `480 | 720 | 1080` stream a fresh on-the-fly transcode that always starts
 *   at 0, so they are driven by a JS virtual timeline instead.
 */
export type VideoQuality = "original" | 480 | 720 | 1080;

/** Delay before the first load so the modal has painted the element. */
const INITIAL_LOAD_DELAY = 100;

export interface PlaybackControllerState {
  isPlaying: boolean;
  isBuffering: boolean;
  playbackRate: number;
  /** Absolute source duration in seconds. */
  duration: number;
  /** Absolute playhead in the source (includes any scrub preview). */
  currentTime: number;
  progress: number;
  bufferedProgress: number;
  error: string | null;
  togglePlay: () => void;
  skip: (seconds: number) => void;
  changeSpeed: (rate: number) => void;
  /** Show a seek target in the UI without committing (native moves live). */
  previewAbsolute: (seconds: number) => void;
  /** Perform the seek at an absolute second, optionally resuming play. */
  commitSeek: (seconds: number, autoplay: boolean) => void;
  scrubStart: () => void;
  scrubEnd: () => void;
  scrubTo: (progressPercent: number) => void;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/**
 * One playback controller for both delivery modes.
 *
 * The UI only ever talks in *absolute* source seconds. Internally:
 *
 *   native (original):  absoluteTime = video.currentTime
 *   compressed:         absoluteTime = streamStartTime + video.currentTime
 *
 * Seeking dispatches to `video.currentTime` in native mode, and to a fresh
 * `?start=` stream in compressed mode. Switching quality hands the current
 * absolute position across to the new mode so playback does not jump.
 */
export function useVideoPlaybackController(
  videoRef: RefObject<HTMLVideoElement | null>,
  fileUrl: string,
  filePath: string,
  isOpen: boolean,
  quality: VideoQuality
): PlaybackControllerState {
  const isNative = quality === "original";

  const [realDuration, setRealDuration] = useState(0);
  const [nativeDuration, setNativeDuration] = useState(0);
  const [nativeTime, setNativeTime] = useState(0);
  const [nativeBuffered, setNativeBuffered] = useState(0);
  const [streamStart, setStreamStart] = useState(0);
  const [engineTime, setEngineTime] = useState(0);
  const [engineBufferedEnd, setEngineBufferedEnd] = useState(0);
  const [previewSeconds, setPreviewSeconds] = useState<number | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isBuffering, setIsBuffering] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1);
  const [error, setError] = useState<string | null>(null);

  // Refs mirror the latest values so stable DOM handlers and the load effect
  // never read a stale render snapshot.
  const isNativeRef = useRef(isNative);
  const streamStartRef = useRef(0);
  const engineTimeRef = useRef(0);
  const nativeTimeRef = useRef(0);
  const realDurationRef = useRef(0);
  const previewRef = useRef<number | null>(null);
  const absoluteTimeRef = useRef(0);
  const wasPlayingBeforeScrub = useRef(false);
  const pendingNativeSeek = useRef<number | null>(null);
  const pendingNativePlay = useRef(false);
  const loadedFileRef = useRef<string | null>(null);
  const prevQualityRef = useRef<VideoQuality>(quality);

  isNativeRef.current = isNative;
  streamStartRef.current = streamStart;
  engineTimeRef.current = engineTime;
  nativeTimeRef.current = nativeTime;
  realDurationRef.current = realDuration;

  /* -------------------- absolute duration from the backend -------------------- */
  useEffect(() => {
    if (!isOpen || !filePath) return;

    let cancelled = false;
    getVideoDuration(filePath)
      .then((res) => {
        if (cancelled) return;
        const duration = res.duration > 0 ? res.duration : 0;
        realDurationRef.current = duration;
        setRealDuration(duration);
      })
      .catch(() => {
        if (!cancelled) setRealDuration(0);
      });

    return () => {
      cancelled = true;
    };
  }, [isOpen, filePath]);

  /* -------------------- media element events -------------------- */
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => { setIsPlaying(true); };
    const onPause = () => { setIsPlaying(false); };
    const onTime = () => {
      if (isNativeRef.current) setNativeTime(video.currentTime);
      else setEngineTime(video.currentTime);
    };
    const onProgress = () => {
      if (video.buffered.length === 0) return;
      const end = video.buffered.end(video.buffered.length - 1);
      if (isNativeRef.current) setNativeBuffered(end);
      else setEngineBufferedEnd(end);
    };
    const onLoadedMetadata = () => {
      if (!isNativeRef.current) return;
      setNativeDuration(video.duration || 0);
      const pending = pendingNativeSeek.current;
      if (pending !== null) {
        pendingNativeSeek.current = null;
        video.currentTime = pending;
      }
      // Deferred play for quality switches: seek first, then start.
      if (pendingNativePlay.current) {
        pendingNativePlay.current = false;
        video.play().catch(() => {
          setIsPlaying(false);
        });
      }
    };
    const onWaiting = () => { setIsBuffering(true); };
    const onPlaying = () => { setIsBuffering(false); };
    const onCanPlay = () => { setIsBuffering(false); };
    // Native seeks (original quality) surface as seeking/seeked.
    const onSeeking = () => { setIsBuffering(true); };
    const onSeeked = () => { setIsBuffering(false); };
    const onEnded = () => { setIsPlaying(false); };
    const onError = () => {
      if (!video.currentSrc) return;
      setError("Unable to load video.");
      setIsBuffering(false);
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("timeupdate", onTime);
    video.addEventListener("progress", onProgress);
    video.addEventListener("loadedmetadata", onLoadedMetadata);
    video.addEventListener("waiting", onWaiting);
    video.addEventListener("playing", onPlaying);
    video.addEventListener("canplay", onCanPlay);
    video.addEventListener("seeking", onSeeking);
    video.addEventListener("seeked", onSeeked);
    video.addEventListener("ended", onEnded);
    video.addEventListener("error", onError);

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("timeupdate", onTime);
      video.removeEventListener("progress", onProgress);
      video.removeEventListener("loadedmetadata", onLoadedMetadata);
      video.removeEventListener("waiting", onWaiting);
      video.removeEventListener("playing", onPlaying);
      video.removeEventListener("canplay", onCanPlay);
      video.removeEventListener("seeking", onSeeking);
      video.removeEventListener("seeked", onSeeked);
      video.removeEventListener("ended", onEnded);
      video.removeEventListener("error", onError);
    };
  }, [isOpen, videoRef]);

  /* -------------------- load a source at an absolute position -------------------- */
  const loadAt = useCallback(
    (targetSeconds: number, autoplay: boolean) => {
      const video = videoRef.current;
      if (!video) return;

      const max = realDurationRef.current;
      const target =
        max > 0 ? clamp(targetSeconds, 0, max) : Math.max(0, targetSeconds);

      previewRef.current = null;
      setPreviewSeconds(null);
      setError(null);
      video.pause();
      // Show the loading spinner immediately on a quality change / restart,
      // before the media element has had a chance to fire waiting.
      setIsBuffering(true);

      if (quality === "original") {
        // Native file: load once, then seek (and play) on metadata so a
        // mid-file quality switch does not flash the first frame.
        pendingNativeSeek.current = target > 0 ? target : null;
        pendingNativePlay.current = target > 0 ? autoplay : false;
        video.src = fileUrl;
        video.load();
        if (target <= 0 && autoplay) {
          video.play().catch(() => {
            setIsPlaying(false);
          });
        }
        return;
      }

      // Compressed: a fresh segment always starts at 0.
      pendingNativeSeek.current = null;
      pendingNativePlay.current = false;
      streamStartRef.current = target;
      setStreamStart(target);
      setEngineTime(0);
      setEngineBufferedEnd(0);
      video.src = buildCompressStreamUrl(fileUrl, target, quality);
      video.load();

      if (autoplay) {
        video.play().catch(() => {
          setIsPlaying(false);
        });
      }
    },
    [fileUrl, quality, videoRef]
  );

  /* -------------------- load on open / file / quality change -------------------- */
  useEffect(() => {
    if (!isOpen || !fileUrl) {
      loadedFileRef.current = null;
      return;
    }

    const isNewFile = loadedFileRef.current !== fileUrl;
    const prevQuality = prevQualityRef.current;

    // On a quality switch, carry the absolute playhead over from the old mode.
    let target = 0;
    if (!isNewFile) {
      target =
        prevQuality === "original"
          ? nativeTimeRef.current
          : streamStartRef.current + engineTimeRef.current;
    }

    loadedFileRef.current = fileUrl;
    prevQualityRef.current = quality;

    if (isNewFile) {
      streamStartRef.current = 0;
      realDurationRef.current = 0;
      setStreamStart(0);
      setEngineTime(0);
      setEngineBufferedEnd(0);
      setNativeTime(0);
      setNativeBuffered(0);
      setNativeDuration(0);
      setRealDuration(0);
    }
    previewRef.current = null;
    setPreviewSeconds(null);
    setError(null);

    const video = videoRef.current;
    const resume = isNewFile ? true : video ? !video.paused : true;

    // Deferred so React StrictMode's double-invoked effect cannot start two
    // loads; the cleanup cancels the first one.
    const timer = setTimeout(
      () => { loadAt(target, resume); },
      isNewFile ? INITIAL_LOAD_DELAY : 0
    );
    return () => { clearTimeout(timer); };
  }, [isOpen, fileUrl, quality, loadAt, videoRef]);

  /* -------------------- derived absolute timeline -------------------- */
  const duration = isNative && nativeDuration > 0 ? nativeDuration : realDuration;
  const durationRef = useRef(0);
  durationRef.current = duration;

  const virtualTime =
    realDuration > 0
      ? Math.min(streamStart + engineTime, realDuration)
      : streamStart + engineTime;
  const absoluteTime = isNative ? nativeTime : virtualTime;
  absoluteTimeRef.current = absoluteTime;

  const displayTime = previewSeconds ?? absoluteTime;
  const progress =
    duration > 0 ? clamp((displayTime / duration) * 100, 0, 100) : 0;

  const bufferedAbsolute = isNative
    ? nativeBuffered
    : realDuration > 0
      ? Math.min(streamStart + engineBufferedEnd, realDuration)
      : 0;
  const bufferedProgress =
    duration > 0 ? clamp((bufferedAbsolute / duration) * 100, 0, 100) : 0;

  /* -------------------- actions -------------------- */
  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      video.play().catch(() => { setIsPlaying(false); });
    } else {
      video.pause();
    }
  }, [videoRef]);

  const previewAbsolute = useCallback(
    (seconds: number) => {
      previewRef.current = seconds;
      setPreviewSeconds(seconds);
      // Native playback can jump immediately and cheaply; compressed cannot.
      if (isNativeRef.current) {
        const video = videoRef.current;
        if (video) video.currentTime = seconds;
      }
    },
    [videoRef]
  );

  const commitSeek = useCallback(
    (seconds: number, autoplay: boolean) => {
      if (quality === "original") {
        const video = videoRef.current;
        if (!video) return;
        previewRef.current = null;
        setPreviewSeconds(null);
        video.currentTime = seconds;
        if (autoplay && video.paused) {
          video.play().catch(() => { setIsPlaying(false); });
        }
        return;
      }
      loadAt(seconds, autoplay);
    },
    [quality, loadAt, videoRef]
  );

  const skip = useCallback(
    (seconds: number) => {
      const video = videoRef.current;
      const base = previewRef.current ?? absoluteTimeRef.current;
      const autoplay = video ? !video.paused : true;
      commitSeek(base + seconds, autoplay);
    },
    [commitSeek, videoRef]
  );

  const changeSpeed = useCallback(
    (rate: number) => {
      const video = videoRef.current;
      if (!video) return;
      video.playbackRate = rate;
      setPlaybackRate(rate);
    },
    [videoRef]
  );

  const scrubStart = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    wasPlayingBeforeScrub.current = !video.paused;
    video.pause();
  }, [videoRef]);

  const scrubTo = useCallback(
    (progressPercent: number) => {
      if (durationRef.current <= 0) return;
      previewAbsolute((progressPercent / 100) * durationRef.current);
    },
    [previewAbsolute]
  );

  const scrubEnd = useCallback(() => {
    const target = previewRef.current;
    if (target === null) return;
    commitSeek(target, wasPlayingBeforeScrub.current);
  }, [commitSeek]);

  return {
    isPlaying,
    isBuffering,
    playbackRate,
    duration,
    currentTime: displayTime,
    progress,
    bufferedProgress,
    error,
    togglePlay,
    skip,
    changeSpeed,
    previewAbsolute,
    commitSeek,
    scrubStart,
    scrubEnd,
    scrubTo,
  };
}
