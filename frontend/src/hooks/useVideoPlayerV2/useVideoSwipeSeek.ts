import { useRef, useState, type RefObject, type TouchEvent } from "react";

/** 1 pixel of horizontal drag maps to this many seconds. */
const SECONDS_PER_PIXEL = 0.04;
/** Delay before a touch is treated as a drag rather than a tap. */
const SCRUB_START_DELAY = 200;

interface UseVideoSwipeSeekParams {
  videoRef: RefObject<HTMLVideoElement | null>;
  duration: number;
  /** Shared flag owned by playback: was the video playing before the scrub? */
  wasPlayingBeforeScrub: RefObject<boolean>;
  /** Seek to an absolute time (clamped by the caller). */
  onSeek: (time: number) => void;
  /** Resume playback after the gesture, if it was playing before. */
  onResume: () => void;
}

/**
 * Horizontal swipe-to-seek on the video surface. A short hold arms scrubbing;
 * `hasScrubbed` is surfaced so the container can ignore the trailing tap.
 */
export function useVideoSwipeSeek({
  videoRef,
  duration,
  wasPlayingBeforeScrub,
  onSeek,
  onResume,
}: UseVideoSwipeSeekParams) {
  const touchStartX = useRef<number | null>(null);
  const touchStartTime = useRef<number | null>(null);
  const isScrubbing = useRef<boolean>(false);
  const scrubTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hasScrubbed = useRef<boolean>(false);
  const [dragDeltaSeconds, setDragDeltaSeconds] = useState<number | null>(null);

  const handleTouchStart = (e: TouchEvent<HTMLDivElement>) => {
    if (e.touches.length === 1) {
      // Deadzone at the bottom 100px so video scrubbing won't conflict with progress bar
      if (e.touches[0].clientY > window.innerHeight - 100) {
        return;
      }

      touchStartX.current = e.touches[0].clientX;
      if (videoRef.current) {
        touchStartTime.current = videoRef.current.currentTime;
      }
      hasScrubbed.current = false;
      isScrubbing.current = false;

      // Small delay before dragging is sensed
      scrubTimeout.current = setTimeout(() => {
        isScrubbing.current = true;
        if (videoRef.current) {
          wasPlayingBeforeScrub.current = !videoRef.current.paused;
          videoRef.current.pause();
        }
      }, SCRUB_START_DELAY);
    }
  };

  const handleTouchMove = (e: TouchEvent<HTMLDivElement>) => {
    if (!isScrubbing.current) return;

    if (
      touchStartX.current === null ||
      touchStartTime.current === null ||
      !videoRef.current
    )
      return;

    hasScrubbed.current = true;
    const deltaX = e.touches[0].clientX - touchStartX.current;

    let newTime = touchStartTime.current + deltaX * SECONDS_PER_PIXEL;

    if (newTime < 0) newTime = 0;
    if (duration > 0 && newTime > duration) newTime = duration;

    setDragDeltaSeconds(newTime - touchStartTime.current);
    onSeek(newTime);
  };

  const handleTouchEnd = () => {
    if (scrubTimeout.current) {
      clearTimeout(scrubTimeout.current);
      scrubTimeout.current = null;
    }

    if (isScrubbing.current && wasPlayingBeforeScrub.current) {
      onResume();
    }

    touchStartX.current = null;
    touchStartTime.current = null;
    isScrubbing.current = false;
    setDragDeltaSeconds(null);

    // Keep hasScrubbed true slightly longer so the click event ignores it
    setTimeout(() => {
      hasScrubbed.current = false;
    }, 100);
  };

  return {
    dragDeltaSeconds,
    hasScrubbed,
    handleTouchStart,
    handleTouchMove,
    handleTouchEnd,
  };
}
