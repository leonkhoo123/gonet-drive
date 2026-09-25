import { useCallback, useEffect, useRef, useState } from "react";

/** How long controls stay visible after the last interaction. */
const CONTROL_TIMEOUT = 5000;

/**
 * Drives the auto-hiding control overlay. Controls are shown on open and while
 * paused, and hidden after a short idle timeout. Pressing a control or hovering
 * the controls/seek bar counts as interacting and pauses the countdown, so the
 * overlay only hides once the pointer has left and nothing is being pressed.
 *
 * Interaction state lives in refs (not state) so the hide timer always reads
 * the live value and the handlers stay referentially stable.
 */
export function useVideoControlsVisibility(isOpen: boolean, isPlaying: boolean) {
  const [showControls, setShowControls] = useState(true);
  const isPressingRef = useRef(false);
  const isHoveringRef = useRef(false);
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearHideTimer = useCallback(() => {
    if (hideTimerRef.current) {
      clearTimeout(hideTimerRef.current);
      hideTimerRef.current = null;
    }
  }, []);

  /** True while the user is pressing a control or hovering the overlay. */
  const isInteracting = useCallback(
    () => isPressingRef.current || isHoveringRef.current,
    []
  );

  const startHideTimer = useCallback(() => {
    clearHideTimer();
    hideTimerRef.current = setTimeout(() => {
      if (!isInteracting()) {
        setShowControls(false);
      }
    }, CONTROL_TIMEOUT);
  }, [clearHideTimer, isInteracting]);

  /** Press (mouse/touch) begins: hold the overlay open. */
  const handlePressStart = useCallback(() => {
    isPressingRef.current = true;
    clearHideTimer();
  }, [clearHideTimer]);

  /** Press ends: resume the idle countdown. */
  const handlePressEnd = useCallback(() => {
    isPressingRef.current = false;
    startHideTimer();
  }, [startHideTimer]);

  /** Pointer enters the controls/seek bar: hold the overlay open. */
  const handleHoverStart = useCallback(() => {
    isHoveringRef.current = true;
    clearHideTimer();
  }, [clearHideTimer]);

  /** Pointer leaves the controls/seek bar: resume the idle countdown. */
  const handleHoverEnd = useCallback(() => {
    isHoveringRef.current = false;
    startHideTimer();
  }, [startHideTimer]);

  /* Show controls when modal opens and start the idle countdown. */
  useEffect(() => {
    if (!isOpen) return;
    setShowControls(true);
    startHideTimer();
    return clearHideTimer;
  }, [isOpen, startHideTimer, clearHideTimer]);

  /* Paused → keep controls visible; playing → resume the countdown. */
  useEffect(() => {
    if (!isPlaying) {
      clearHideTimer();
      setShowControls(true);
    } else {
      startHideTimer();
    }
  }, [isPlaying, startHideTimer, clearHideTimer]);

  return {
    showControls,
    setShowControls,
    clearHideTimer,
    startHideTimer,
    handlePressStart,
    handlePressEnd,
    handleHoverStart,
    handleHoverEnd,
  };
}
