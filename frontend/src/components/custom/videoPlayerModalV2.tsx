/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
  Play,
  Pause,
  Zap,
  SkipBack,
  SkipForward,
  LogOut,
  TextCursorInput,
  ListX,
  RotateCw,
} from "lucide-react";
import type { FileInterface } from "@/api/api-file";

import { useDialogHistory } from "@/hooks/useDialogHistory";

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

const CONTROL_TIMEOUT = 2500;

const VideoPlayerModalV2: React.FC<VideoPlayerModalProps> = ({
  file,
  isOpen,
  onClose,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  /* -------------------- video states -------------------- */
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1.0);
  const [progress, setProgress] = useState(0);
  const [bufferedProgress, setBufferedProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);

  /* -------------------- ui states -------------------- */
  const [showControls, setShowControls] = useState(true);
  const [isInteracting, setIsInteracting] = useState(false);
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /* -------------------- rename / flags -------------------- */
  const [newName, setNewname] = useState("");
  const [isNewName, setisNewname] = useState(false);
  const [disqualified, setDisqualified] = useState(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [tempName, setTempName] = useState("");

  /* -------------------- rotation -------------------- */
  const [rotation, setRotation] = useState(0);
  const [isRotation, setisRotation] = useState(false);

  const isSideways = rotation === 90 || rotation === 270;

  /* -------------------- swipe to seek -------------------- */
  const touchStartX = useRef<number | null>(null);
  const touchStartTime = useRef<number | null>(null);
  const isScrubbing = useRef<boolean>(false);
  const scrubTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hasScrubbed = useRef<boolean>(false);
  const [dragDeltaSeconds, setDragDeltaSeconds] = useState<number | null>(null);

  /* =====================================================
     CONTROL VISIBILITY CORE (IMPORTANT)
     ===================================================== */

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

  /* =====================================================
     VIDEO TAP BEHAVIOR
     ===================================================== */

  const handleVideoTap = () => {
    if (hasScrubbed.current) return;
    
    if (showControls) {
      clearHideTimer();
      setShowControls(false);
    } else {
      setShowControls(true);
      startHideTimer();
    }
  };

  const handleTouchStart = (e: React.TouchEvent<HTMLDivElement>) => {
    if (e.touches.length === 1) {
      touchStartX.current = e.touches[0].clientX;
      if (videoRef.current) {
        touchStartTime.current = videoRef.current.currentTime;
      }
      hasScrubbed.current = false;
      isScrubbing.current = false;

      // Small delay before dragging is sensed (200ms)
      scrubTimeout.current = setTimeout(() => {
        isScrubbing.current = true;
      }, 200);
    }
  };

  const handleTouchMove = (e: React.TouchEvent<HTMLDivElement>) => {
    if (!isScrubbing.current) return;
    
    if (
      touchStartX.current === null ||
      touchStartTime.current === null ||
      !videoRef.current
    ) return;

    hasScrubbed.current = true;
    const deltaX = e.touches[0].clientX - touchStartX.current;
    
    // Sensitivity: 1 pixel = 0.04 seconds
    const SECONDS_PER_PIXEL = 0.04;
    let newTime = touchStartTime.current + deltaX * SECONDS_PER_PIXEL;

    if (newTime < 0) newTime = 0;
    if (duration > 0 && newTime > duration) newTime = duration;

    setDragDeltaSeconds(newTime - touchStartTime.current);

    videoRef.current.currentTime = newTime;
    setCurrentTime(newTime);
    if (duration > 0) {
      setProgress((newTime / duration) * 100);
    }
  };

  const handleTouchEnd = () => {
    if (scrubTimeout.current) {
      clearTimeout(scrubTimeout.current);
      scrubTimeout.current = null;
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

  /* =====================================================
     CONTROL BAR INTERACTION LOCK
     ===================================================== */

  const handleControlPressStart = () => {
    setIsInteracting(true);
    clearHideTimer();
  };

  const handleControlPressEnd = () => {
    setIsInteracting(false);
    startHideTimer();
  };

  /* =====================================================
     VIDEO LOAD / EVENTS
     ===================================================== */

  useEffect(() => {
    if (!isOpen) return;

    const timer = setTimeout(() => {
      const video = videoRef.current;
      if (!video) return;

      const vidUrl = file.url;

      video.src = vidUrl;
      video.play().catch(() => { setIsPlaying(false); });
    }, 100);

    return () => { clearTimeout(timer); };
  }, [isOpen, file.url]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => { setIsPlaying(true); };
    const onPause = () => { setIsPlaying(false); };
    const onLoaded = () => { setDuration(video.duration || 0); };
    const onTime = () => {
      setCurrentTime(video.currentTime);
      setProgress((video.currentTime / video.duration) * 100);
    };
    const onProgress = () => {
      if (!video.duration || video.buffered.length === 0) return;
      const end = video.buffered.end(video.buffered.length - 1);
      setBufferedProgress((end / video.duration) * 100);
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("loadedmetadata", onLoaded);
    video.addEventListener("timeupdate", onTime);
    video.addEventListener("progress", onProgress);

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("loadedmetadata", onLoaded);
      video.removeEventListener("timeupdate", onTime);
      video.removeEventListener("progress", onProgress);
    };
  }, []);

  /* =====================================================
     ACTIONS
     ===================================================== */

  const togglePlay = useCallback(() => {
    const v = videoRef.current;
    if (!v) return;
    if (isPlaying) {
      v.pause();
    } else {
      void v.play();
    }
  },[isPlaying]);

  const skip = useCallback((sec: number) => {
    const v = videoRef.current;
    if (v) v.currentTime += sec;
  },[]);

  const changeSpeed = (rate: number) => {
    const v = videoRef.current;
    if (!v) return;
    v.playbackRate = rate;
    setPlaybackRate(rate);
  };

  const handleRotation = () => {
    const currRotation = (rotation + 90) % 360
    setRotation(currRotation);
    setisRotation(true);
    if (currRotation === 0) {
      setisRotation(false);
    }
  };

  /* =====================================================
     RENAME
     ===================================================== */

  const handleRenameSave = useCallback(() => {
    let finalName = tempName.trim();
    const ext = file.name.includes(".")
      ? file.name.substring(file.name.lastIndexOf("."))
      : "";

    if (!finalName.includes(".") && ext) finalName += ext;

    if (finalName !== file.name) {
      setNewname(finalName);
      setisNewname(true);
    }

    setShowRenameModal(false);
  }, [tempName, file.name]);

  const handleRenameDefault = () => {
    setNewname("");
    setisNewname(false);
    setShowRenameModal(false);
  };

  const handleRenameCancel = useCallback(() => { setShowRenameModal(false); },[]);

  // --- Rename Modal Logic ---
  const openRenameModal = () => {
    setTempName(newName);
    setShowRenameModal(true);
  };

  const handleDisqualified = () => {
    setisNewname(false);
    setNewname("");
    setDisqualified(!disqualified);
  };

  /* =====================================================
     BACK BUTTON
     ===================================================== */
  const handleDismiss = useCallback((): void => {
    if (showRenameModal) {
      // Logic for canceling rename
      handleRenameCancel();
    } else {
      // Logic for closing the file/modal
      onClose(false, file.path, false, newName, 0);
    }
  }, [showRenameModal, file.path, newName, handleRenameCancel, onClose]);
  
  useDialogHistory(isOpen, handleDismiss);

  /* =====================================================
    KEYBOARD LISTENER
    ===================================================== */

  const handleKeyPress = useCallback((event: KeyboardEvent) => {

    // The key property returns the character pressed
    const key: string = event.key;
    // shiftKey property is a boolean indicating if Shift was held
    const isShift: boolean = event.shiftKey;

    let actionDescription = '';

    switch (key) {
      case 'ArrowLeft':
        if (showRenameModal) {
          return; //dont do anything in rename
        }
        if (isShift) {
          actionDescription = 'skip(-3)';
          skip(-3)
        } else {
          actionDescription = 'skip(-1)';
          skip(-1)
        }
        break;

      case 'ArrowRight':
        if (showRenameModal) {
          return; //dont do anything in rename
        }
        if (isShift) {
          actionDescription = 'skip(3)';
          skip(3)
        } else {
          actionDescription = 'skip(1)';
          skip(1)
        }
        break;

      case ' ': // Spacebar
        if (showRenameModal) {
          return; //dont do anything in rename
        }
        actionDescription = 'togglePlay';
        togglePlay()
        break;

      case 'Enter':
        if (showRenameModal) {
          actionDescription = 'handleRenameSave';
          handleRenameSave()
        }
        break;

      case 'Escape':
        actionDescription = 'handleDismiss';
        // handleDismiss();
        window.history.back();
        break;
      default:
        // Ignore other keys
        return;
    }

    // Prevent default browser actions for navigation/scrolling keys when not editing name
    if (!showRenameModal && (key === 'ArrowLeft' || key === 'ArrowRight' || key === ' ')) {
      event.preventDefault();
    }
    console.log(`Key Pressed: ${actionDescription}`);

  }, [showRenameModal, togglePlay, skip, handleRenameSave]);

  useEffect(() => {
    // Add the keydown listener to the document
    document.addEventListener('keydown', handleKeyPress as (e: Event) => void);
    // Cleanup: Remove the event listener when the component unmounts
    return () => {
      document.removeEventListener('keydown', handleKeyPress as (e: Event) => void);
    };
  }, [handleKeyPress]);

  /* =====================================================
     RENDER
     ===================================================== */

  if (!isOpen) return null;

  const formatTime = (t: number) =>
    `${Math.floor(t / 60)}:${String(Math.floor(t % 60)).padStart(2, "0")}`;


  return (
    <div className="fixed inset-0 bg-black z-10 select-none">
      {/* VIDEO AREA */}
      <div
        className="flex items-center justify-center h-full touch-none"
        onClick={handleVideoTap}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
        onTouchCancel={handleTouchEnd}
      >
        <video
          ref={videoRef}
          autoPlay
          playsInline
          preload="auto"
          className={`${isSideways ? "max-h-[100dvw] max-w-[100dvh]" : "max-h-dvh max-w-full"} object-contain`}
          style={{ transform: `rotate(${rotation}deg)` }}
        />
      </div>

      {/* DRAG DELTA POPUP */}
      {dragDeltaSeconds !== null && (
        <div className="absolute top-12 left-1/2 -translate-x-1/2 bg-black/60 text-white px-4 py-2 rounded-full font-bold text-lg tracking-wider pointer-events-none z-20">
          {dragDeltaSeconds >= 0 ? "+" : "-"}{formatTime(Math.abs(dragDeltaSeconds))}
        </div>
      )}

      {/* --- Time Display --- */}
      <div className="absolute bottom-2 left-1 w-full text-left text-white text-sm select-none">
        <span>
          {formatTime(currentTime)} / {formatTime(duration)}
        </span>
        <span className="text-gray-400 ml-2">
          (-{formatTime(duration - currentTime)})
        </span>

        {disqualified ? (
          <span className="ml-3 text-red-500 font-bold">Disqualified</span>
        ) : (
          ""
        )}
        {isNewName ? (
          <span className="ml-3 text-green-500 font-bold">{newName}</span>
        ) : (
          <span className="ml-3 text-white/40 font-bold">{file.name}</span>
        )}
        {isRotation ? (
          <span className="ml-3 text-green-500 font-bold">({rotation}°)</span>
        ) : (
          ""
        )}
      </div>

      {/* PROGRESS */}
      <div
        className={`absolute w-full transition-all duration-300 bg-gray-700/50 ${
          showControls ? "bottom-12 h-3" : "bottom-0 h-2"
        }`}
      >
        <div
          className="absolute h-full bg-gray-500 pointer-events-none"
          style={{ width: `${bufferedProgress}%` }}
        />
        <div
          className="absolute h-full bg-white/70 pointer-events-none"
          style={{ width: `${progress}%` }}
        />
        
        {/* Invisible range input for native dragging/scrubbing */}
        <input
          type="range"
          min={0}
          max={100}
          step="any"
          value={progress || 0}
          onChange={(e) => {
            const v = videoRef.current;
            if (!v || !duration) return;
            
            const newProgress = parseFloat(e.target.value);
            const newTime = (newProgress / 100) * duration;
            
            v.currentTime = newTime; 
            
            setProgress(newProgress);
            setCurrentTime(newTime);
          }}
          className="absolute inset-0 w-full h-full opacity-0 cursor-pointer m-0"
        />
      </div>

      {/* CONTROLS */}
      <div
        className={`absolute right-0 top-0 bottom-12 flex flex-col overflow-y-auto p-2 transition-opacity duration-300 ${showControls ? "opacity-100" : "opacity-0 pointer-events-none"
          }`}
        onMouseDown={handleControlPressStart}
        onMouseUp={handleControlPressEnd}
        onTouchStart={handleControlPressStart}
        onTouchEnd={handleControlPressEnd}
      >
        <div className="my-auto flex flex-col space-y-2 min-h-min w-24">
          <Button
            variant="ghost"
            onClick={() => { skip(-1); }}
            // If your parent uses onMouseDown/onTouchStart, 
            // you must stop those specifically too:
            className="hover:bg-white/80 w-full bg-white/30"
          >
            <SkipBack className="h-4 w-4 mr-1" /> 1s
          </Button>

        {/* forward 3 sec */}
        <Button
          variant="ghost"
          onClick={() => { skip(3); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* forward 1 sec */}
        <Button
          variant="ghost"
          onClick={() => { skip(1); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* speed x2  */}
        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 2.0); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x2"}
        </Button>

        {/* slow mo x0.25 */}
        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 0.25); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x0.25"}
        </Button>

        {/* play/pause */}
        <div className="text-sm text-white text-center">{playbackRate}x</div>
        <Button
          variant="ghost"
          size="icon"
          onClick={togglePlay}
          className="hover:bg-white/80 w-full bg-white/30 py-10"
        >
          {isPlaying ? (
            <Pause className="h-5 w-5" />
          ) : (
            <Play className="h-5 w-5" />
          )}
        </Button>

        {/* --- Rename Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={openRenameModal}
          className="hover:bg-green-300/80 w-full bg-green-300/30 mt-5 landscape:mt-2"
        >
          <TextCursorInput className="h-5 w-5" />
        </Button>

        {/* --- disqualified Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleDisqualified}
          className="hover:bg-red-300/80 w-full bg-red-300/30 mt-5 landscape:mt-2"
        >
          <ListX className="h-5 w-5" />
        </Button>

        {/* --- Rotation Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleRotation}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <RotateCw className="h-5 w-5" />
        </Button>

        {/* --- close --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => { onClose(disqualified, file.path, isNewName, newName, rotation); }}
          className="hover:bg-white/80 w-full bg-white/30 mt-5 landscape:mt-2"
        >
          <LogOut className="h-5 w-5" />
        </Button>
        </div>
      </div>

      {/* RENAME MODAL */}
      {showRenameModal && (
        <div className="absolute inset-0 bg-black/70 flex items-center justify-center">
          <div className="bg-white/70 rounded-md p-4 w-full max-w-5xl shadow-lg text-black mx-2">
            <h3 className="font-semibold mb-2">Rename File</h3>
            <input
              type="text"
              value={tempName}
              onChange={(e) => { setTempName(e.target.value); }}
              className="w-full border-2 border-white/30 p-2 rounded mb-4"
              placeholder="New Video Name"
              autoFocus
            />
            <div className="flex justify-between">
              <Button
                onClick={handleRenameDefault}
                className="bg-gray-200/80 hover:bg-gray-300 text-black"
              >
                Default
              </Button>
              <div className="space-x-2">
                <Button
                  onClick={handleRenameCancel}
                  className="bg-gray-300/80 hover:bg-gray-400 text-black"
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleRenameSave}
                  className="bg-primary/80 hover:bg-primary text-primary-foreground"
                >
                  Save
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default VideoPlayerModalV2;
