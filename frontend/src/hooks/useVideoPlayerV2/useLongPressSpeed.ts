import { useCallback, useRef, useState, type RefObject, type TouchEvent } from "react";

/** Hold duration before the speed burst kicks in. */
const LONG_PRESS_DELAY = 300;
/** Movement (px) that cancels a pending long press. */
const MOVE_CANCEL_PX = 12;

interface UseLongPressSpeedParams {
  videoRef: RefObject<HTMLVideoElement | null>;
  /** The user's chosen playback rate, restored when the burst ends. */
  baseRate: number;
  /** Temporary multiplier applied while held. */
  burstRate?: number;
}

/**
 * Press-and-hold to temporarily speed up playback (default 2x), restoring the
 * chosen rate on release. A quick tap or a drag never triggers it, and
 * `wasLongPress` lets the caller swallow the trailing click so the tap does not
 * also toggle the controls.
 *
 * This replaces swipe-to-seek in the compressed player, where dragging cannot
 * scrub live and every commit restarts the transcode.
 */
export function useLongPressSpeed({
  videoRef,
  baseRate,
  burstRate = 2,
}: UseLongPressSpeedParams) {
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const active = useRef(false);
  const startX = useRef<number | null>(null);
  const startY = useRef<number | null>(null);
  const moved = useRef(false);
  const baseRateRef = useRef(baseRate);
  baseRateRef.current = baseRate;
  const wasLongPress = useRef(false);
  const [isBurst, setIsBurst] = useState(false);

  const clearTimer = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
  }, []);

  const stopBurst = useCallback(() => {
    if (!active.current) return;
    active.current = false;
    const video = videoRef.current;
    if (video) video.playbackRate = baseRateRef.current;
    setIsBurst(false);
  }, [videoRef]);

  const handleTouchStart = (e: TouchEvent<HTMLDivElement>) => {
    if (e.touches.length !== 1) return;
    // Leave the bottom strip (progress bar) and controls alone.
    if (e.touches[0].clientY > window.innerHeight - 100) return;

    wasLongPress.current = false;
    moved.current = false;
    startX.current = e.touches[0].clientX;
    startY.current = e.touches[0].clientY;

    clearTimer();
    timer.current = setTimeout(() => {
      timer.current = null;
      if (moved.current) return;
      const video = videoRef.current;
      if (!video) return;
      active.current = true;
      wasLongPress.current = true;
      video.playbackRate = burstRate;
      setIsBurst(true);
    }, LONG_PRESS_DELAY);
  };

  const handleTouchMove = (e: TouchEvent<HTMLDivElement>) => {
    if (startX.current === null || startY.current === null) return;
    const dx = e.touches[0].clientX - startX.current;
    const dy = e.touches[0].clientY - startY.current;
    if (Math.hypot(dx, dy) > MOVE_CANCEL_PX) {
      moved.current = true;
      // A drag is not a long press: cancel an arming burst.
      if (!active.current) clearTimer();
    }
  };

  const handleTouchEnd = () => {
    clearTimer();
    startX.current = null;
    startY.current = null;
    moved.current = false;
    stopBurst();
  };

  return {
    isBurst,
    wasLongPress,
    handleTouchStart,
    handleTouchMove,
    handleTouchEnd,
  };
}
