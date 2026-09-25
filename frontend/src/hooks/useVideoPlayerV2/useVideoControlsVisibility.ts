import { useCallback, useEffect, useRef, useState } from "react";

/** How long controls stay visible after the last interaction. */
const CONTROL_TIMEOUT = 2500;

/**
 * Drives the auto-hiding control overlay. Controls are shown on open and while
 * paused, hidden after a short idle timeout, and kept visible while the user is
 * pressing a control.
 */
export function useVideoControlsVisibility(isOpen: boolean, isPlaying: boolean) {
  const [showControls, setShowControls] = useState(true);
  const [isInteracting, setIsInteracting] = useState(false);
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearHideTimer = useCallback(() => {
    if (hideTimerRef.current) {
      clearTimeout(hideTimerRef.current);
      hideTimerRef.current = null;
    }
  }, []);

  const startHideTimer = useCallback(() => {
    clearHideTimer();
    hideTimerRef.current = setTimeout(() => {
      if (!isInteracting) {
        setShowControls(false);
      }
    }, CONTROL_TIMEOUT);
  }, [clearHideTimer, isInteracting]);

  /* Show controls when modal opens */
  useEffect(() => {
    if (!isOpen) return;
    setShowControls(true);
    startHideTimer();
    return clearHideTimer;
  }, [isOpen, startHideTimer, clearHideTimer]);

  /* Pause → keep controls visible */
  useEffect(() => {
    if (!isPlaying) {
      clearHideTimer();
      setShowControls(true);
    } else {
      startHideTimer();
    }
  }, [isPlaying, startHideTimer, clearHideTimer]);

  const handleControlPressStart = () => {
    setIsInteracting(true);
    clearHideTimer();
  };

  const handleControlPressEnd = () => {
    setIsInteracting(false);
    startHideTimer();
  };

  return {
    showControls,
    setShowControls,
    clearHideTimer,
    startHideTimer,
    handleControlPressStart,
    handleControlPressEnd,
  };
}
