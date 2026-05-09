import React, { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import {
  Play,
  Pause,
  Volume2,
  VolumeX,
  X,
  Maximize,
  Minimize,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { FileInterface } from "@/api/api-file";
import { useForceDarkStatusBar } from "@/hooks/useForceDarkStatusBar";

const SWIPE_CLOSE_THRESHOLD = 100; // px vertical to trigger close
const SWIPE_CLOSE_VELOCITY = 0.5; // px/ms flick threshold
const CLOSE_ANIM_DURATION = 200; // ms

interface VideoPlayerModalProps {
  file: FileInterface;
  isOpen: boolean;
  onClose: (
    isDisqualified: boolean,
    oriPath: string,
    isNewName: boolean,
    newName: string,
    rotation: number
  ) => void;
}

const formatTime = (time: number) => {
  if (isNaN(time)) return "00:00";
  const mins = Math.floor(time / 60);
  const secs = Math.floor(time % 60);
  return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
};

const VideoPlayerModalGeneric: React.FC<VideoPlayerModalProps> = ({
  file,
  isOpen,
  onClose,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // ---- Swipe-down close (mobile) ----
  const [isClosing, setIsClosing] = useState(false);
  const closeAnimRef = useRef(false);
  const swipeHandled = useRef(false);
  const touchStartX = useRef(0);
  const touchStartY = useRef(0);
  const touchStartTime = useRef(0);

  const [isPlaying, setIsPlaying] = useState(false);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [showControls, setShowControls] = useState(true);
  const [playbackRate, setPlaybackRate] = useState<number>(1);

  useForceDarkStatusBar(isOpen);
  const [isInteracting, setIsInteracting] = useState(false);
  const isInteractingRef = useRef(false);

  const hideControlsTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearHideTimer = useCallback(() => {
    if (hideControlsTimeoutRef.current) {
      clearTimeout(hideControlsTimeoutRef.current);
      hideControlsTimeoutRef.current = null;
    }
  }, []);

  const startHideTimer = useCallback(() => {
    clearHideTimer();
    hideControlsTimeoutRef.current = setTimeout(() => {
      if (!isInteractingRef.current) {
        setShowControls(false);
      }
    }, 3000);
  }, [clearHideTimer]);

  /* Show controls on open, start 3s timer */
  useEffect(() => {
    if (!isOpen) return;
    setShowControls(true);
    startHideTimer();
    return clearHideTimer;
  }, [isOpen, startHideTimer, clearHideTimer]);

  /* Reset swipe-close state on open */
  useEffect(() => {
    if (isOpen) {
      setIsClosing(false);
      closeAnimRef.current = false;
    }
  }, [isOpen]);

  /* Pause → keep controls. Play → start timer. */
  useEffect(() => {
    if (!isPlaying) {
      clearHideTimer();
      setShowControls(true);
    } else {
      startHideTimer();
    }
  }, [isPlaying, startHideTimer, clearHideTimer]);

  /* =====================================================
     MOUSE / TOUCH ON VIDEO AREA
     ===================================================== */

  const handleMouseMove = useCallback(() => {
    if (!isInteracting) {
      setShowControls(true);
      startHideTimer();
    }
  }, [isInteracting, startHideTimer]);

  const handleMouseLeave = useCallback(() => {
    if (isPlaying && !isInteracting) {
      setShowControls(false);
    }
  }, [isPlaying, isInteracting]);

  const handleVideoTap = useCallback(() => {
    if (showControls) {
      clearHideTimer();
      setShowControls(false);
    } else {
      setShowControls(true);
      startHideTimer();
    }
  }, [showControls, clearHideTimer, startHideTimer]);

  /* =====================================================
     CONTROL BAR INTERACTION LOCK
     ===================================================== */

  const handleControlPressStart = useCallback(() => {
    isInteractingRef.current = true;
    setIsInteracting(true);
    clearHideTimer();
  }, [clearHideTimer]);

  const handleControlPressEnd = useCallback(() => {
    isInteractingRef.current = false;
    setIsInteracting(false);
    startHideTimer();
  }, [startHideTimer]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !isOpen) return;

    const onPlay = () => { setIsPlaying(true); };
    const onPause = () => { setIsPlaying(false); };
    const onLoadedMetadata = () => { setDuration(video.duration); };
    const onTimeUpdate = () => {
      setCurrentTime(video.currentTime);
      setProgress((video.currentTime / (video.duration || 1)) * 100);
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("loadedmetadata", onLoadedMetadata);
    video.addEventListener("timeupdate", onTimeUpdate);

    // Auto-play
    video.play().catch((err: unknown) => { console.log("Auto-play prevented", err); });

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("loadedmetadata", onLoadedMetadata);
      video.removeEventListener("timeupdate", onTimeUpdate);
    };
  }, [isOpen, file.url]);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => { document.removeEventListener("fullscreenchange", handleFullscreenChange); };
  }, []);

  const togglePlay = useCallback(() => {
    if (videoRef.current) {
      if (!videoRef.current.paused) {
        videoRef.current.pause();
      } else {
        void videoRef.current.play();
      }
    }
  }, []);


  const handleVolumeChange = useCallback((newVolume: number[]) => {
    const val = newVolume[0];
    setVolume(val);
    if (videoRef.current) {
      videoRef.current.volume = val;
      if (val > 0 && videoRef.current.muted) {
        setIsMuted(false);
        videoRef.current.muted = false;
      } else if (val === 0 && !videoRef.current.muted) {
        setIsMuted(true);
        videoRef.current.muted = true;
      }
    }
  }, []);

  const toggleMute = useCallback(() => {
    if (videoRef.current) {
      const newMutedState = !videoRef.current.muted;
      videoRef.current.muted = newMutedState;
      setIsMuted(newMutedState);
      if (!newMutedState && videoRef.current.volume === 0) {
        handleVolumeChange([0.5]);
      }
    }
  }, [handleVolumeChange]);

  const handleClose = useCallback(() => {
    if (videoRef.current) {
      videoRef.current.pause();
    }
    onClose(false, file.path, false, "", 0);
  }, [file.path, onClose]);

  /* =====================================================
     SWIPE-DOWN CLOSE (MOBILE)
     ===================================================== */

  const triggerSwipeClose = useCallback(() => {
    if (closeAnimRef.current) return;
    closeAnimRef.current = true;
    setIsClosing(true);
    setTimeout(() => {
      handleClose();
    }, CLOSE_ANIM_DURATION);
  }, [handleClose]);

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
    touchStartY.current = e.touches[0].clientY;
    touchStartTime.current = Date.now();
    swipeHandled.current = false;
  }, []);

  const handleTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      if (swipeHandled.current) return;
      const deltaX = e.changedTouches[0].clientX - touchStartX.current;
      const deltaY = e.changedTouches[0].clientY - touchStartY.current;
      const elapsed = Date.now() - touchStartTime.current;

      // Vertical swipe-down close (mobile only)
      if (deltaY > SWIPE_CLOSE_THRESHOLD && Math.abs(deltaY) > Math.abs(deltaX)) {
        swipeHandled.current = true;
        triggerSwipeClose();
        return;
      }
      if (deltaY > 0 && elapsed > 0 && deltaY / elapsed > SWIPE_CLOSE_VELOCITY && Math.abs(deltaY) > Math.abs(deltaX)) {
        swipeHandled.current = true;
        triggerSwipeClose();
        return;
      }
    },
    [triggerSwipeClose]
  );

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;

      // Prevent default browser behaviors for these keys if we're handling them
      if (e.key === " " || e.key === "Escape" || e.key.toLowerCase() === "m") {
        if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
          return; // Ignore if typing in an input
        }
      }

      if (e.key === "Escape") {
        e.stopPropagation();
      }

      switch (e.key) {
        case " ":
          e.preventDefault();
          e.stopPropagation();
          e.stopImmediatePropagation();
          togglePlay();
          break;
        case "Escape":
          e.preventDefault();
          e.stopPropagation();
          e.stopImmediatePropagation();
          handleClose();
          break;
        case "m":
        case "M":
          e.preventDefault();
          e.stopPropagation();
          e.stopImmediatePropagation();
          toggleMute();
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown, { capture: true });
    return () => { window.removeEventListener("keydown", handleKeyDown, { capture: true }); };
  }, [isOpen, togglePlay, handleClose, toggleMute]);

  if (!isOpen) return null;

  const handleProgressChange = (newProgress: number[]) => {
    const val = newProgress[0];
    if (videoRef.current && duration) {
      const newTime = (val / 100) * duration;
      videoRef.current.currentTime = newTime;
      setProgress(val);
    }
  };

  const handleSpeedChange = (speed: number) => {
    setPlaybackRate(speed);
    if (videoRef.current) {
      videoRef.current.playbackRate = speed;
    }
  };

  const toggleFullscreen = () => {
    if (!containerRef.current) return;

    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen().catch((err: unknown) => {
        console.error("Error attempting to enable fullscreen:", err);
      });
    } else {
      void document.exitFullscreen();
    }
  };



  return (
    <div
      role="dialog"
      className={`fixed inset-0 z-[100] flex items-center justify-center bg-black/90 backdrop-blur-sm transition-transform duration-200 ease-out ${
        isClosing ? "translate-y-full" : "translate-y-0"
      }`}
      style={{ touchAction: "none" }}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      <div
        ref={containerRef}
        className="relative flex w-full max-w-5xl flex-col items-center justify-center overflow-hidden rounded-xl bg-black shadow-2xl"
        onMouseLeave={handleMouseLeave}
        onMouseMove={handleMouseMove}
      >
        <video
          ref={videoRef}
          src={file.url}
          className="max-h-[85vh] w-full cursor-pointer bg-black object-contain"
          onClick={handleVideoTap}
          controls={false}
          playsInline
        />

        {/* Top bar with close button */}
        <div
          className={`absolute left-0 top-0 flex w-full items-center justify-between bg-gradient-to-b from-black/80 to-transparent p-4 transition-opacity duration-300 ${
            showControls ? "opacity-100" : "opacity-0 pointer-events-none"
          }`}
          onMouseDown={handleControlPressStart}
          onMouseUp={handleControlPressEnd}
          onTouchStart={handleControlPressStart}
          onTouchEnd={handleControlPressEnd}
        >
          <div className="flex-1 truncate pr-4 text-sm font-medium text-white shadow-black drop-shadow-md">
            {file.name}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 rounded-full bg-black/40 text-white hover:bg-black/60"
            onClick={handleClose}
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Bottom controls */}
        <div
          className={`absolute bottom-0 left-0 w-full bg-gradient-to-t from-black/90 via-black/60 to-transparent p-4 transition-opacity duration-300 ${
            showControls ? "opacity-100" : "opacity-0 pointer-events-none"
          }`}
          onMouseDown={handleControlPressStart}
          onMouseUp={handleControlPressEnd}
          onTouchStart={handleControlPressStart}
          onTouchEnd={handleControlPressEnd}
        >
          {/* Progress bar */}
          <div className="mb-4 flex items-center gap-3">
            <span className="text-xs font-medium text-white w-10 text-right">
              {formatTime(currentTime)}
            </span>
            <Slider
              value={[progress]}
              max={100}
              step={0.1}
              onValueChange={handleProgressChange}
              className="flex-1 cursor-pointer"
            />
            <span className="text-xs font-medium text-white/70 w-10">
              {formatTime(duration)}
            </span>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Button
                variant="ghost"
                size="icon"
                className="h-10 w-10 rounded-full text-white hover:bg-white/20 hover:text-white"
                onClick={togglePlay}
              >
                {isPlaying ? (
                  <Pause className="h-5 w-5 fill-current" />
                ) : (
                  <Play className="h-5 w-5 fill-current ml-1" />
                )}
              </Button>

              <div className="flex items-center gap-2 group">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-white hover:bg-white/20 hover:text-white"
                  onClick={toggleMute}
                >
                  {isMuted || volume === 0 ? (
                    <VolumeX className="h-5 w-5" />
                  ) : (
                    <Volume2 className="h-5 w-5" />
                  )}
                </Button>
                <div className="w-0 overflow-hidden transition-all duration-300 group-hover:w-24 opacity-0 group-hover:opacity-100">
                  <Slider
                    value={[isMuted ? 0 : volume]}
                    max={1}
                    step={0.01}
                    onValueChange={handleVolumeChange}
                    className="w-24 cursor-pointer"
                  />
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    className="h-8 px-2 text-sm font-medium text-white hover:bg-white/20 hover:text-white"
                  >
                    {playbackRate}x
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-32 bg-black/90 text-white border-white/10 z-[150]">
                  {[0.25, 0.5, 0.75, 1, 1.25, 1.5, 1.75, 2].map((speed) => (
                    <DropdownMenuItem
                      key={speed}
                      className={`cursor-pointer hover:bg-white/20 focus:bg-white/20 ${
                        playbackRate === speed ? "bg-white/10 font-bold" : ""
                      }`}
                      onClick={() => { handleSpeedChange(speed); }}
                    >
                      {speed}x
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>

              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-white hover:bg-white/20 hover:text-white"
                onClick={toggleFullscreen}
              >
                {isFullscreen ? (
                  <Minimize className="h-5 w-5" />
                ) : (
                  <Maximize className="h-5 w-5" />
                )}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default VideoPlayerModalGeneric;
