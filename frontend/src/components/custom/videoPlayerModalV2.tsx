import React, { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
  Play,
  Pause,
  Zap,
  SkipBack,
  SkipForward,
  ChevronsLeft,
  ChevronsRight,
  LogOut,
  TextCursorInput,
  ListX,
  RotateCw,
} from "lucide-react";
import type { FileInterface } from "@/api/api-file";
import axiosLayer from "@/api/axiosLayer";

import { useDialogHistory } from "@/hooks/useDialogHistory";
import { useForceDarkStatusBar } from "@/hooks/useForceDarkStatusBar";

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

/** A detected event as an absolute [start, end] pair in seconds. */
type EventSpan = [number, number];

const METADATA_DIRNAME = ".vid_metadata";

/**
 * Read the event array out of a timestamps JSON payload.
 * Mirrors the tolerant parsing of the Python player: accepts
 * `{"events": [...]}`, the compact `{"e": [...]}`, or a bare array.
 */
const extractEventArray = (data: unknown): unknown[] => {
  if (Array.isArray(data)) return data;
  if (data && typeof data === "object") {
    const obj = data as Record<string, unknown>;
    if (Array.isArray(obj.events)) return obj.events;
    if (Array.isArray(obj.e)) return obj.e;
  }
  return [];
};

/** Normalise raw JSON entries into sorted, clamped [start, end] spans. */
const parseEventSpans = (data: unknown): EventSpan[] => {
  const spans: EventSpan[] = [];

  for (const item of extractEventArray(data)) {
    let start: number;
    let end: number;

    if (Array.isArray(item) && item.length >= 2) {
      start = Number(item[0]);
      end = Number(item[1]);
    } else if (item && typeof item === "object") {
      const obj = item as Record<string, unknown>;
      start = Number(obj.start);
      end = Number(obj.end);
    } else {
      continue;
    }

    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    spans.push([Math.min(start, end), Math.max(start, end)]);
  }

  spans.sort((a, b) => a[0] - b[0]);
  return spans;
};

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

  /* -------------------- event metadata -------------------- */
  const [events, setEvents] = useState<EventSpan[]>([]);

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

  useForceDarkStatusBar(isOpen);

  /* -------------------- swipe to seek -------------------- */
  const touchStartX = useRef<number | null>(null);
  const touchStartTime = useRef<number | null>(null);
  const isScrubbing = useRef<boolean>(false);
  const scrubTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hasScrubbed = useRef<boolean>(false);
  const wasPlayingBeforeScrub = useRef<boolean>(false);
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

      // Small delay before dragging is sensed (200ms)
      scrubTimeout.current = setTimeout(() => {
        isScrubbing.current = true;
        if (videoRef.current) {
          wasPlayingBeforeScrub.current = !videoRef.current.paused;
          videoRef.current.pause();
        }
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

    if (isScrubbing.current && videoRef.current && wasPlayingBeforeScrub.current) {
      videoRef.current.play().catch(() => { setIsPlaying(false); });
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

  /* Load the sibling metadata JSON written by the AI pipeline:
     <video folder>/.vid_metadata/<video filename>_timestamps.json
     (e.g. clip.mp4_timestamps.json - the extension is kept in the name).
     A missing or malformed file is non-fatal - the player just shows no markers. */
  useEffect(() => {
    if (!isOpen || !file.path) {
      setEvents([]);
      return;
    }

    const slash = file.path.lastIndexOf("/");
    const dir = slash >= 0 ? file.path.slice(0, slash) : "";
    const metaPath = `${dir}/${METADATA_DIRNAME}/${file.name}_timestamps.json`;

    let cancelled = false;
    setEvents([]);

    axiosLayer
      .get<unknown>(`/user/document/read/file${encodeURI(metaPath)}`)
      .then((res) => {
        if (cancelled) return;
        const payload: unknown =
          typeof res.data === "string" ? (JSON.parse(res.data) as unknown) : res.data;
        setEvents(parseEventSpans(payload));
      })
      .catch(() => {
        if (!cancelled) setEvents([]);
      });

    return () => { cancelled = true; };
  }, [isOpen, file.path, file.name]);

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

  /** Jump to the next detected event start after the playhead. */
  const nextEvent = useCallback(() => {
    const v = videoRef.current;
    if (!v) return;
    const next = events.find(([start]) => start > v.currentTime + 0.05);
    if (next) v.currentTime = next[0];
  }, [events]);

  /** Jump to the previous detected event start before the playhead. */
  const prevEvent = useCallback(() => {
    const v = videoRef.current;
    if (!v) return;
    let candidate: number | null = null;
    for (const [start] of events) {
      if (start < v.currentTime - 0.05) candidate = start;
      else break;
    }
    if (candidate !== null) v.currentTime = candidate;
  }, [events]);

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

      case '<':
      case ',': // Shift+comma / comma → previous event start
        if (showRenameModal) {
          return;
        }
        actionDescription = 'prevEvent';
        prevEvent();
        break;

      case '>':
      case '.': // Shift+period / period → next event start
        if (showRenameModal) {
          return;
        }
        actionDescription = 'nextEvent';
        nextEvent();
        break;

      case 'n':
      case 'N':
        if (showRenameModal) {
          return;
        }
        actionDescription = 'nextEvent';
        nextEvent();
        break;

      case 'p':
      case 'P':
        if (showRenameModal) {
          return;
        }
        actionDescription = 'prevEvent';
        prevEvent();
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

  }, [showRenameModal, togglePlay, skip, nextEvent, prevEvent, handleRenameSave]);

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
    `${String(Math.floor(t / 60))}:${String(Math.floor(t % 60)).padStart(2, "0")}`;


  return (
    <div className="fixed inset-0 bg-black z-[100] select-none">
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
          className={`w-full h-full object-contain`}
          style={{ transform: `rotate(${String(rotation)}deg)` }}
        />
      </div>

      {/* DRAG DELTA POPUP */}
      {dragDeltaSeconds !== null && (
        <div className="absolute top-12 left-1/2 -translate-x-1/2 bg-black/60 text-white px-4 py-2 rounded-full font-bold text-lg tracking-wider pointer-events-none z-20">
          {dragDeltaSeconds >= 0 ? "+" : "-"}{formatTime(Math.abs(dragDeltaSeconds))}
        </div>
      )}

      {/* BOTTOM BACKGROUND FOR VISIBILITY */}
      <div
        className={`absolute bottom-0 left-0 w-full transition-all duration-300 pointer-events-none bg-black/60 ${
          showControls ? "h-12 opacity-100" : "h-0 opacity-0"
        }`}
      />

      {/* --- Time Display --- */}
      <div className="absolute bottom-2 left-1 w-full text-left text-white text-sm select-none z-10">
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
      {/* z-30 keeps the seek bar above the time display (z-10) and the
          controls column, so the full width — including the far left and
          the area under the buttons — stays clickable for scrubbing. */}
      <div
        className={`absolute w-full transition-all duration-300 bg-gray-700/50 z-30 ${
          showControls ? "bottom-12 h-3" : "bottom-0 h-2"
        }`}
      >
        <div
          className="absolute h-full bg-gray-500 pointer-events-none"
          style={{ width: `${String(bufferedProgress)}%` }}
        />
        <div
          className="absolute h-full bg-white/70 pointer-events-none"
          style={{ width: `${String(progress)}%` }}
        />

        {/* Detected-event markers loaded from .vid_metadata/<video>_timestamps.json.
            A faint emerald tint with a crisp accent line; brighter and glowing
            while the playhead sits inside an event. */}
        {duration > 0 && events.length > 0 && (
          <div className="absolute inset-0 pointer-events-none">
            {events.map(([start, end]) => {
              const left = Math.min(100, Math.max(0, (start / duration) * 100));
              const right = Math.min(100, Math.max(left, (end / duration) * 100));
              const width = Math.max(right - left, 0.4);
              const isInside = currentTime >= start && currentTime <= end;
              return (
                <div
                  key={`${String(start)}-${String(end)}`}
                  className={`absolute top-0 bottom-0 rounded-[1px] border-b-2 transition-all duration-200 ${
                    isInside
                      ? "bg-emerald-300/35 border-emerald-200 shadow-[0_0_8px_rgba(52,211,153,0.85)]"
                      : "bg-emerald-400/15 border-emerald-400/70"
                  }`}
                  style={{ left: `${String(left)}%`, width: `${String(width)}%` }}
                />
              );
            })}
          </div>
        )}
        
        {/* Invisible range input for native dragging/scrubbing */}
        <input
          type="range"
          min={0}
          max={100}
          step="any"
          value={progress || 0}
          onPointerDown={(e) => {
            e.stopPropagation();
            if (videoRef.current) {
              wasPlayingBeforeScrub.current = !videoRef.current.paused;
              videoRef.current.pause();
            }
          }}
          onPointerUp={(e) => {
            e.stopPropagation();
            if (videoRef.current && wasPlayingBeforeScrub.current) {
              videoRef.current.play().catch(() => { setIsPlaying(false); });
            }
          }}
          onPointerCancel={(e) => {
            e.stopPropagation();
            if (videoRef.current && wasPlayingBeforeScrub.current) {
              videoRef.current.play().catch(() => { setIsPlaying(false); });
            }
          }}
          onChange={(e) => {
            const v = videoRef.current;
            if (!v || !duration) return;
            
            const newProgress = parseFloat(e.target.value);
            const newTime = (newProgress / 100) * duration;
            
            v.currentTime = newTime; 
            
            setProgress(newProgress);
            setCurrentTime(newTime);
          }}
          className={`absolute left-0 w-full opacity-0 cursor-pointer m-0 ${
            showControls ? "top-0 h-[60px]" : "inset-0 h-full"
          }`}
        />
      </div>

      {/* CONTROLS */}
      <div
        className={`absolute right-0 top-0 bottom-12 flex flex-col p-1 lg:p-2 transition-opacity duration-300 ${showControls ? "opacity-100" : "opacity-0 pointer-events-none"
          }`}
        onMouseDown={handleControlPressStart}
        onMouseUp={handleControlPressEnd}
        onTouchStart={handleControlPressStart}
        onTouchEnd={handleControlPressEnd}
      >
        <div className="m-auto flex flex-col gap-1 lg:gap-2 h-full max-h-[100%] overflow-y-auto scrollbar-hide w-20 lg:w-24 py-2 justify-center">
          <Button
            variant="ghost"
            onClick={() => { skip(-1); }}
            // If your parent uses onMouseDown/onTouchStart, 
            // you must stop those specifically too:
            className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
          >
            <SkipBack className="h-4 w-4 mr-1" /> 1s
          </Button>

        {/* forward 3 sec */}
        <Button
          variant="ghost"
          onClick={() => { skip(3); }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* forward 1 sec */}
        <Button
          variant="ghost"
          onClick={() => { skip(1); }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* previous/next detected event - only rendered when metadata exists */}
        {events.length > 0 && (
          <>
            <Button
              variant="ghost"
              onClick={prevEvent}
              title="Previous event (P or Shift+,)"
              className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
            >
              <ChevronsLeft className="h-4 w-4 mr-1" /> Evt
            </Button>

            <Button
              variant="ghost"
              onClick={nextEvent}
              title="Next event (N or Shift+.)"
              className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
            >
              Evt <ChevronsRight className="h-4 w-4 ml-1" />
            </Button>
          </>
        )}

        {/* slow mo x0.25 */}
        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 0.25); }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x0.25"}
        </Button>

        {/* play/pause */}
        <div className="text-sm text-white text-center flex items-center justify-center min-h-[20px] shrink-0">{playbackRate}x</div>
        <Button
          variant="ghost"
          size="icon"
          onClick={togglePlay}
          className="hover:bg-white/80 w-full bg-white/30 flex-[2] min-h-[40px] max-h-24"
        >
          {isPlaying ? (
            <Pause className="h-6 w-6" />
          ) : (
            <Play className="h-6 w-6" />
          )}
        </Button>

        {/* --- Rename Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={openRenameModal}
          className="hover:bg-green-300/80 w-full bg-green-300/30 flex-1 min-h-[32px] max-h-12"
        >
          <TextCursorInput className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>

        {/* --- disqualified Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleDisqualified}
          className="hover:bg-red-300/80 w-full bg-red-300/30 flex-1 min-h-[32px] max-h-12"
        >
          <ListX className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>

        {/* --- Rotation Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleRotation}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12"
        >
          <RotateCw className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>

        {/* --- close --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => { onClose(disqualified, file.path, isNewName, newName, rotation); }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12"
        >
          <LogOut className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>
        </div>
      </div>

      {/* RENAME MODAL */}
      {showRenameModal && (
        <div className="absolute inset-0 z-40 bg-black/70 flex items-center justify-center">
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
