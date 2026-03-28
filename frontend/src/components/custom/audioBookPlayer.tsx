import { useEffect, useRef, useState, useCallback } from "react";
import { X, Play, Pause, Volume2, VolumeX, Headphones, RotateCcw, RotateCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { toast } from "sonner";
import TrackInfo from "./trackInfo";
import { reportAudioBookProgress } from "@/api/api-audiobook";
import type { AudioBookItem } from "@/api/api-audiobook";

interface AudioBookPlayerProps {
  file: AudioBookItem | null;
  onClose: () => void;
  forcePause?: boolean;
}

function formatTime(seconds: number) {
  if (isNaN(seconds)) return "00:00";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) {
    return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

export function AudioBookPlayer({ file, onClose, forcePause }: AudioBookPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const playerContainerRef = useRef<HTMLDivElement>(null);
  const [currentFile, setCurrentFile] = useState<AudioBookItem | null>(file);
  const [isVisible, setIsVisible] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [allowBackground, setAllowBackground] = useState(true);
  const [isVolumeOpen, setIsVolumeOpen] = useState(false);
  const [isTimeModalOpen, setIsTimeModalOpen] = useState(false);
  const [timeInput, setTimeInput] = useState("");

  const lastSavedTime = useRef<number>(0);
  const syncInterval = useRef<number | null>(null);

  const saveProgress = useCallback(() => {
    if (currentFile && audioRef.current) {
      const time = audioRef.current.currentTime;
      if (Math.abs(time - lastSavedTime.current) > 1) { // Only save if changed
        const nameWithoutAb = currentFile.name.startsWith("ab_") ? currentFile.name.slice(3) : currentFile.name;
        reportAudioBookProgress(nameWithoutAb, time).catch(console.error);
        lastSavedTime.current = time;
      }
    }
  }, [currentFile]);

  useEffect(() => {
    if (file) {
      if (!currentFile || currentFile.url !== file.url) {
        if (currentFile) {
          saveProgress(); // Save old file progress before switching
        }
        setCurrentFile(file);
        setCurrentTime(file.progress_time || 0);
        lastSavedTime.current = file.progress_time || 0;
        requestAnimationFrame(() => { setIsVisible(true); });
      } else {
        setIsVisible(true);
      }
    } else {
      if (currentFile) {
        saveProgress(); // Save old file progress on close
      }
      setIsVisible(false);
      setIsPlaying(false);
      if (audioRef.current) audioRef.current.pause();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [file]);

  useEffect(() => {
    if (forcePause) setIsPlaying(false);
  }, [forcePause]);

  useEffect(() => {
    if (audioRef.current) {
      if (isPlaying) {
        audioRef.current.play().catch((e: unknown) => { console.error("Error playing audio:", e); });
      } else {
        audioRef.current.pause();
      }
    }
  }, [isPlaying, currentFile]);

  // Sync progress every 3 minutes
  useEffect(() => {
    if (isPlaying) {
      syncInterval.current = window.setInterval(saveProgress, 3 * 60 * 1000);
    } else if (syncInterval.current) {
      window.clearInterval(syncInterval.current);
    }
    return () => {
      if (syncInterval.current) window.clearInterval(syncInterval.current);
    };
  }, [isPlaying, saveProgress]);

  // Handle visibility change for syncing and background play
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden) {
        saveProgress(); // Sync when minimized
        if (!allowBackground && audioRef.current && !audioRef.current.paused) {
          audioRef.current.pause();
          setIsPlaying(false);
        }
      }
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => { document.removeEventListener("visibilitychange", handleVisibilityChange); };
  }, [allowBackground, saveProgress]);

  // Set media session metadata
  useEffect(() => {
    if ('mediaSession' in navigator && currentFile) {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: currentFile.name.startsWith("ab_") ? currentFile.name.slice(3) : currentFile.name,
        artist: 'Audio Book',
      });
      navigator.mediaSession.setActionHandler('play', () => { setIsPlaying(true); });
      navigator.mediaSession.setActionHandler('pause', () => { setIsPlaying(false); });
      navigator.mediaSession.setActionHandler('seekbackward', handlePrev);
      navigator.mediaSession.setActionHandler('seekforward', handleNext);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentFile]);

  // Auto-play when file changes
  useEffect(() => {
    if (currentFile && audioRef.current) {
      audioRef.current.currentTime = currentFile.progress_time || 0;
      setIsPlaying(true);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentFile?.url]);

  const handleTimeSubmit = () => {
    const parts = timeInput.split(':').map(Number);
    let seconds = 0;
    if (parts.length === 3) {
      seconds = parts[0] * 3600 + parts[1] * 60 + parts[2];
    } else if (parts.length === 2) {
      seconds = parts[0] * 60 + parts[1];
    } else if (parts.length === 1) {
      seconds = parts[0];
    } else {
      toast.error("Invalid time format. Use HH:MM:SS or MM:SS");
      return;
    }

    if (isNaN(seconds)) {
      toast.error("Invalid time format. Use HH:MM:SS or MM:SS");
      return;
    }

    if (audioRef.current) {
      let newTime = seconds;
      const maxTime = duration || (currentFile?.total_length ?? 0);
      if (newTime > maxTime) newTime = maxTime;
      if (newTime < 0) newTime = 0;
      audioRef.current.currentTime = newTime;
      setCurrentTime(newTime);
      setIsTimeModalOpen(false);
      setIsPlaying(true);
    }
  };

  const openTimeModal = () => {
    setTimeInput(formatTime(currentTime));
    setIsTimeModalOpen(true);
  };

  const handleNext = () => {
    if (audioRef.current) {
      let newTime = audioRef.current.currentTime + 3;
      if (newTime > duration) newTime = duration;
      audioRef.current.currentTime = newTime;
      setCurrentTime(newTime);
    }
  };

  const handlePrev = () => {
    if (audioRef.current) {
      let newTime = audioRef.current.currentTime - 5;
      if (newTime < 0) newTime = 0;
      audioRef.current.currentTime = newTime;
      setCurrentTime(newTime);
    }
  };

  const togglePlay = () => { setIsPlaying(!isPlaying); };

  const handleTimeUpdate = () => {
    if (audioRef.current) setCurrentTime(audioRef.current.currentTime);
  };

  const handleLoadedMetadata = () => {
    if (audioRef.current) setDuration(audioRef.current.duration);
  };

  const handleSeek = (value: number[]) => {
    if (audioRef.current) {
      audioRef.current.currentTime = value[0];
      setCurrentTime(value[0]);
    }
  };

  const handleVolumeChange = (value: number[]) => {
    const newVolume = value[0];
    setVolume(newVolume);
    if (audioRef.current) audioRef.current.volume = newVolume;
    setIsMuted(newVolume === 0);
  };

  const toggleMute = () => {
    if (audioRef.current) {
      const newMutedState = !isMuted;
      audioRef.current.muted = newMutedState;
      setIsMuted(newMutedState);
      if (!newMutedState && volume === 0) {
        setVolume(1);
        audioRef.current.volume = 1;
      }
    }
  };

  const handleEnded = () => {
    setIsPlaying(false);
    setCurrentTime(0);
    saveProgress();
  };

  const handleClose = () => {
    saveProgress();
    onClose();
  };

  if (!currentFile) return null;

  return (
    <div
      ref={playerContainerRef}
      className={`overflow-visible min-h-0 flex flex-col w-full bg-background border-border shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] shrink-0 z-50 transition-all duration-500 ease-in-out relative ${isVisible ? "grid-rows-[1fr] opacity-100 border-t" : "grid-rows-[0fr] opacity-0 border-t-0"}`}
    >
      <div className={`w-full p-3 md:px-6 flex flex-col transition-all duration-700 ${isVisible ? 'visible opacity-100' : 'invisible opacity-0'}`}>
        <div className="absolute top-0 left-0 right-0 -translate-y-1/2 px-0 z-10">
          <Slider
            value={[currentTime]}
            max={duration || currentFile.total_length || 100}
            step={1}
            onValueChange={handleSeek}
            className="w-full h-1"
          />
        </div>

        <audio
          ref={audioRef}
          src={currentFile.url}
          onTimeUpdate={handleTimeUpdate}
          onLoadedMetadata={handleLoadedMetadata}
          onEnded={handleEnded}
          onPause={saveProgress}
          className="hidden"
        />

        <div className="flex flex-row items-center justify-between gap-1 md:gap-4 w-full">
          {/* Left: Controls and Time */}
          <div className="flex items-center gap-2 md:gap-4 shrink-0 w-auto md:w-[30%]">
            <div className="flex items-center gap-0.5 md:gap-2">
              <Button variant="ghost" size="icon" className="w-8 h-8 rounded-full shrink-0" onClick={handlePrev} title="Backward 5s">
                <RotateCcw className="w-4 h-4" />
              </Button>
              <Button variant="secondary" size="icon" className="w-9 h-9 md:w-10 md:h-10 rounded-full shrink-0" onClick={togglePlay}>
                {isPlaying ? <Pause className="w-4 h-4 md:w-5 md:h-5" /> : <Play className="w-4 h-4 md:w-5 md:h-5 ml-1" />}
              </Button>
              <Button variant="ghost" size="icon" className="w-8 h-8 rounded-full shrink-0" onClick={handleNext} title="Forward 3s">
                <RotateCw className="w-4 h-4" />
              </Button>
            </div>
            <span className="text-xs text-muted-foreground hidden md:block whitespace-nowrap">
              {formatTime(currentTime)} / {formatTime(duration || currentFile.total_length)}
            </span>
          </div>

          {/* Middle: Track Info */}
          <div className="flex-1 min-w-0 flex justify-center cursor-default">
            <TrackInfo text={currentFile.name.startsWith("ab_") ? currentFile.name.slice(3) : currentFile.name} currentTime={currentTime} duration={duration || currentFile.total_length} onClick={openTimeModal} />
          </div>

          {/* Right: Extra Controls */}
          <div className="flex items-center justify-end gap-1 md:gap-4 shrink-0 w-auto md:w-[30%]">
            <div className="hidden md:flex items-center gap-2 bg-muted/50 p-1.5 rounded-md">
              <Switch
                id="bg-play-audiobook"
                checked={allowBackground}
                onCheckedChange={(checked) => {
                  setAllowBackground(checked);
                  toast.success(`Background Play ${checked ? "Enabled" : "Disabled"}`);
                }}
                className="scale-75 data-[state=checked]:bg-primary"
              />
              <Label htmlFor="bg-play-audiobook" className="text-[10px] font-medium leading-none cursor-pointer">
                BG Play
              </Label>
            </div>

            <div className="hidden md:flex items-center gap-2 w-28">
              <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={toggleMute}>
                {isMuted || volume === 0 ? <VolumeX className="w-4 h-4" /> : <Volume2 className="w-4 h-4" />}
              </Button>
              <Slider value={[isMuted ? 0 : volume]} max={1} step={0.01} onValueChange={handleVolumeChange} className="w-full" />
            </div>

            <div className="flex md:hidden items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 shrink-0"
                onClick={() => {
                  const newState = !allowBackground;
                  setAllowBackground(newState);
                  toast.success(`Background Play ${newState ? "Enabled" : "Disabled"}`);
                }}
              >
                <Headphones className={`w-4 h-4 ${allowBackground ? "text-primary" : "text-muted-foreground"}`} />
              </Button>
              <Popover open={isVolumeOpen} onOpenChange={setIsVolumeOpen}>
                <PopoverTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={(e) => { if (isVolumeOpen) { e.preventDefault(); toggleMute(); } }} onPointerDown={(e) => { if (isVolumeOpen) e.preventDefault(); }}>
                    {isMuted || volume === 0 ? <VolumeX className="w-4 h-4" /> : <Volume2 className="w-4 h-4" />}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-12 h-32 p-2 mb-2 bg-background flex justify-center" side="top" align="center">
                  <Slider orientation="vertical" value={[isMuted ? 0 : volume]} max={1} step={0.01} onValueChange={handleVolumeChange} className="h-full" />
                </PopoverContent>
              </Popover>
            </div>

            <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0 md:ml-2 hover:bg-destructive/10 hover:text-destructive" onClick={handleClose}>
              <X className="w-5 h-5" />
            </Button>
          </div>
        </div>
      </div>

      <Dialog open={isTimeModalOpen} onOpenChange={setIsTimeModalOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Skip to Time</DialogTitle>
          </DialogHeader>
          <div className="flex items-center space-x-2 py-4">
            <div className="grid flex-1 gap-2">
              <Label htmlFor="time" className="sr-only">
                Time
              </Label>
              <Input
                id="time"
                value={timeInput}
                onChange={(e) => { setTimeInput(e.target.value); }}
                placeholder="HH:MM:SS or MM:SS"
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    handleTimeSubmit();
                  }
                }}
                autoFocus
              />
              <span className="text-xs text-muted-foreground pl-1">Format: HH:MM:SS (e.g., 01:30:00)</span>
            </div>
          </div>
          <DialogFooter className="sm:justify-end">
            <Button
              type="button"
              variant="secondary"
              onClick={() => { setIsTimeModalOpen(false); }}
            >
              Cancel
            </Button>
            <Button type="button" onClick={handleTimeSubmit}>
              OK
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
