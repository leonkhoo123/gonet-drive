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
  onClose: (isDisqualified: boolean, oriPath: string, isNewName: boolean, newName: string, rotation: number) => void;
}

const VideoPlayerModal: React.FC<VideoPlayerModalProps> = ({
  file,
  isOpen,
  onClose,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1.0);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const [newName, setNewname] = useState<string>("");
  const [isNewName, setisNewname] = useState<boolean>(false);
  const [disqualified, setDisqualified] = useState<boolean>(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [tempName, setTempName] = useState<string>("");
  const [bufferedProgress, setBufferedProgress] = useState(0); // Percentage
  const [rotation, setRotation] = useState(0); //degree 
  const [isRotation, setisRotation] = useState<boolean>(false); //degree 
  const [showControls, setShowControls] = useState(true);
  const controlsTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const controlTimeout = 2500;

  const startHideTimer = useCallback(() => {
    if (controlsTimeoutRef.current) {
      clearTimeout(controlsTimeoutRef.current);
    }
    controlsTimeoutRef.current = setTimeout(() => {
      setShowControls(false);
    }, controlTimeout);
  }, []);

  const handleInteractionStart = useCallback(() => {
    if (showControls) {
      setShowControls(false);
    } else {
      setShowControls(true);
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current);
      }
    }
  }, [showControls]);

  const handleInteractionEnd = useCallback(() => {
    if (showControls) {
      startHideTimer();
    }
  }, [showControls, startHideTimer]);

  useEffect(() => {
    startHideTimer();
    return () => {
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current);
      }
    };
  }, [startHideTimer]);

  const formatTime = (time: number): string => {
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
    return `${minutes}:${seconds.toString().padStart(2, "0")}`;
  };

  useEffect(() => {
    if (!isOpen) return;
    const timer = setTimeout(() => {
      const video = videoRef.current;
      if (video) {
        const vidUrl = file.url;
        video.src = vidUrl;
        video.play().catch(() => { setIsPlaying(false); });
      }
    }, 100);
    return () => { clearTimeout(timer); };
  }, [isOpen, file.url]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const handlePlay = () => { setIsPlaying(true); };
    const handlePause = () => { setIsPlaying(false); };
    const handleLoadedMetadata = () => { setDuration(video.duration || 0); };
    const handleTimeUpdate = () => {
      if (!video.duration) return;
      setCurrentTime(video.currentTime);
      setProgress((video.currentTime / video.duration) * 100);
    };

    // --- New Handler for Buffered Progress ---
    const handleProgress = () => {
      if (!video.duration || video.buffered.length === 0) return;

      // The 'buffered' property is a TimeRanges object.
      // We want the end time of the *last* buffered range.
      const lastBufferedTime = video.buffered.end(video.buffered.length - 1);
      const bufferedPercent = (lastBufferedTime / video.duration) * 100;
      setBufferedProgress(bufferedPercent);
    };
    // ----------------------------------------

    video.addEventListener("play", handlePlay);
    video.addEventListener("pause", handlePause);
    video.addEventListener("loadedmetadata", handleLoadedMetadata);
    video.addEventListener("timeupdate", handleTimeUpdate);
    video.addEventListener("buffered", handleTimeUpdate);
    video.addEventListener("progress", handleProgress);

    return () => {
      video.removeEventListener("play", handlePlay);
      video.removeEventListener("pause", handlePause);
      video.removeEventListener("loadedmetadata", handleLoadedMetadata);
      video.removeEventListener("timeupdate", handleTimeUpdate);
      video.removeEventListener("progress", handleProgress);
    };
  }, []);

  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    if (isPlaying) video.pause();
    else void video.play();
  }, [isPlaying]);

  const skip = useCallback((seconds: number) => {
    const video = videoRef.current;
    if (video) video.currentTime += seconds;
  }, []);

  const changeSpeed = useCallback((rate: number) => {
    const video = videoRef.current;
    if (video) {
      video.playbackRate = rate;
      setPlaybackRate(rate);
    }
  }, []);

  const handleSeek = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      const video = videoRef.current;
      if (!video || !duration) return;
      const rect = e.currentTarget.getBoundingClientRect();
      const percent = (e.clientX - rect.left) / rect.width;
      video.currentTime = percent * duration;
      setProgress(percent * 100);
    },
    [duration]
  );

  const handleDisqualified = () => {
    setisNewname(false);
    setNewname("");
    setDisqualified(!disqualified);
  };

  const handleRotation = () => {
    const currRotation = (rotation + 90) % 360
    setRotation(currRotation);
    setisRotation(true);
    if (currRotation === 0) {
      setisRotation(false);
    }
  };

  // Helper function to determine if video is rotated sideways
  const isSideways = rotation === 90 || rotation === 270;


  // --- Rename Modal Logic ---
  const openRenameModal = () => {
    setTempName(newName);
    setShowRenameModal(true);
  };

  const handleRenameCancel = () => { setShowRenameModal(false); };

  const handleRenameSave = useCallback(() => {
    let finalName = tempName.trim();
    const originalExt = file.name.includes(".")
      ? file.name.substring(file.name.lastIndexOf("."))
      : "";

    // If user didn't type extension, append the original one
    if (!finalName.includes(".") && originalExt) {
      finalName += originalExt;
    }

    // Only update if name changed
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

  if (!isOpen) return null;

  {/* keyboard listener */ }
  const handleDismiss = useCallback((): void => {
    if (showRenameModal) {
      // Logic for canceling rename
      handleRenameCancel();
    } else {
      // Logic for closing the file/modal
      onClose(false, file.path, false, newName, 0);
    }
  }, [showRenameModal, file.path, newName, handleRenameCancel, onClose]);

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

  // The dependency array is empty, ensuring the function is stable
  useDialogHistory(isOpen, handleDismiss);

  useEffect(() => {
    // Add the keydown listener to the document
    document.addEventListener('keydown', handleKeyPress as (e: Event) => void);
    // Cleanup: Remove the event listener when the component unmounts
    return () => {
      document.removeEventListener('keydown', handleKeyPress as (e: Event) => void);
    };
  }, [handleKeyPress]);

  return (
    <div className="fixed inset-0 z-10 bg-black select-none"
      onMouseDown={handleInteractionStart}
      onMouseUp={handleInteractionEnd}
      onTouchStart={handleInteractionStart}
      onTouchEnd={handleInteractionEnd}>

      <div className="flex items-center justify-center bg-black h-full">
        <video
          preload="auto"
          ref={videoRef}
          controls={false}
          autoPlay
          playsInline
          className={`
        ${isSideways ? 'max-h-[100dvw] max-w-[100dvh]' : 'max-h-dvh max-w-full'}
        object-contain
        transition-transform duration-300
      `}
          // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
          style={{ transform: `rotate(${rotation}deg)` }}
        />
      </div>


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

      {/* --- Progress Bar ---*/}
      <div
        className="absolute bottom-0 w-full h-2 bg-gray-700 cursor-pointer"
        onClick={handleSeek}
      >
        {/* 💡 1. Buffered Progress (White) */}
        <div
          className="h-2 bg-white/20 absolute top-0 left-0 transition-all"
          // You will need a state variable, e.g., 'bufferedProgress', to set this width
          // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
          style={{ width: `${bufferedProgress}%` }}
        />

        {/* ✅ 2. Main Current Time Progress (Blue) */}
        <div
          className="h-2 bg-primary transition-all absolute top-0 left-0"
          // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* --- Control Bar --- */}
      <div className={`absolute bottom-1/8 right-0 p-2 bg-transparent rounded-md flex flex-col items-center justify-center sm:w-[100px] w-[70px] space-y-2 transition-opacity duration-500 ${showControls ? "opacity-100" : "opacity-0 pointer-events-none"}`}>
        {/* back 1 sec */}
        <Button
          variant="ghost"
          onClick={()=>{skip(-1); }}
          // If your parent uses onMouseDown/onTouchStart, 
          // you must stop those specifically too:
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <SkipBack className="h-4 w-4 mr-1" /> 1s
        </Button>

        {/* forward 3 sec */}
        <Button
          variant="ghost"
          onClick={() => { skip(3); }}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* forward 1 sec */}
        <Button
          variant="ghost"
          onClick={() => { skip(1); }}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* speed x2  */}
        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 2.0); }}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x2"}
        </Button>

        {/* speed x3 */}
        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 3.0); }}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x3"}
        </Button>

        {/* play/pause */}
        <div className="text-sm text-white">{playbackRate.toFixed(1)}x</div>
        <Button
          variant="ghost"
          size="icon"
          onClick={togglePlay}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
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
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-green-300/80 w-full bg-green-300/30 mt-5"
        >
          <TextCursorInput className="h-5 w-5" />
        </Button>

        {/* --- disqualified Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleDisqualified}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-red-300/80 w-full bg-red-300/30 mt-5"
        >
          <ListX className="h-5 w-5" />
        </Button>

        {/* --- Rotation Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleRotation}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <RotateCw className="h-5 w-5" />
        </Button>

        {/* --- close --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => { onClose(disqualified, file.path, isNewName, newName, rotation); }}
          onMouseDown={(e) => { e.stopPropagation(); }}
          onTouchStart={(e) => { e.stopPropagation(); }}
          className="hover:bg-white/80 w-full bg-white/30 mt-5"
        >
          <LogOut className="h-5 w-5" />
        </Button>

        {/* --- close --- */}
        {/* <Button
          variant="ghost"
          size="icon"
          onClick={handleFullscreen}
          className="hover:bg-white/80 w-full bg-white/30 mt-10"
        >
          <Maximize className="h-5 w-5" />
        </Button> */}
      </div>

      {/* --- Rename Modal --- */}
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

export default VideoPlayerModal;
