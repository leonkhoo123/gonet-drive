import React, { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import {
  Play,
  Pause,
  Volume2,
  VolumeX,
  X,
  Music,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { FileInterface } from "@/api/api-file";

interface MusicPreviewModalProps {
  file: FileInterface | null;
  isOpen: boolean;
  onClose: () => void;
}

const formatTime = (time: number) => {
  if (isNaN(time)) return "00:00";
  const mins = Math.floor(time / 60);
  const secs = Math.floor(time % 60);
  return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
};

export const MusicPreviewModal: React.FC<MusicPreviewModalProps> = ({
  file,
  isOpen,
  onClose,
}) => {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio || !isOpen || !file) return;

    const onPlay = () => { setIsPlaying(true); };
    const onPause = () => { setIsPlaying(false); };
    const onLoadedMetadata = () => { setDuration(audio.duration); };
    const onTimeUpdate = () => {
      setCurrentTime(audio.currentTime);
      setProgress((audio.currentTime / (audio.duration || 1)) * 100);
    };

    audio.addEventListener("play", onPlay);
    audio.addEventListener("pause", onPause);
    audio.addEventListener("loadedmetadata", onLoadedMetadata);
    audio.addEventListener("timeupdate", onTimeUpdate);

    // Auto-play
    audio.play().catch((err: unknown) => { console.log("Auto-play prevented", err); });

    return () => {
      audio.removeEventListener("play", onPlay);
      audio.removeEventListener("pause", onPause);
      audio.removeEventListener("loadedmetadata", onLoadedMetadata);
      audio.removeEventListener("timeupdate", onTimeUpdate);
    };
  }, [isOpen, file]);

  const togglePlay = useCallback(() => {
    if (audioRef.current) {
      if (!audioRef.current.paused) {
        audioRef.current.pause();
      } else {
        void audioRef.current.play();
      }
    }
  }, []);

  const handleVolumeChange = useCallback((newVolume: number[]) => {
    const val = newVolume[0];
    setVolume(val);
    if (audioRef.current) {
      audioRef.current.volume = val;
      if (val > 0 && audioRef.current.muted) {
        setIsMuted(false);
        audioRef.current.muted = false;
      } else if (val === 0 && !audioRef.current.muted) {
        setIsMuted(true);
        audioRef.current.muted = true;
      }
    }
  }, []);

  const toggleMute = useCallback(() => {
    if (audioRef.current) {
      const newMutedState = !audioRef.current.muted;
      audioRef.current.muted = newMutedState;
      setIsMuted(newMutedState);
      if (!newMutedState && audioRef.current.volume === 0) {
        handleVolumeChange([0.5]);
      }
    }
  }, [handleVolumeChange]);

  const handleClose = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause();
    }
    onClose();
  }, [onClose]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;

      if (e.key === " " || e.key === "Escape" || e.key.toLowerCase() === "m") {
        if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
          return;
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

  if (!isOpen || !file) return null;

  const handleProgressChange = (newProgress: number[]) => {
    const val = newProgress[0];
    if (audioRef.current && duration) {
      const newTime = (val / 100) * duration;
      audioRef.current.currentTime = newTime;
      setProgress(val);
    }
  };

  const handleSpeedChange = (speed: number) => {
    setPlaybackRate(speed);
    if (audioRef.current) {
      audioRef.current.playbackRate = speed;
    }
  };

  return (
    <div
      role="dialog"
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/90 backdrop-blur-sm p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) { handleClose(); }
      }}
    >
      <div className="relative flex w-full max-w-2xl flex-col overflow-hidden rounded-xl bg-black shadow-2xl border border-white/10">
        <audio
          ref={audioRef}
          src={file.url}
          className="hidden"
          controls={false}
        />

        {/* Top bar with close button */}
        <div className="flex w-full items-center justify-between p-4 bg-white/5">
          <div className="flex flex-col flex-1 truncate pr-4">
             <span className="text-sm font-medium text-white/70">Now Playing</span>
             <span className="text-base font-bold text-white truncate">{file.name}</span>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 rounded-full text-white hover:bg-white/20"
            onClick={handleClose}
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Visualizer/Icon Area */}
        <div 
          className="w-full aspect-video flex items-center justify-center bg-gradient-to-br from-white/5 to-white/10"
          onClick={togglePlay}
        >
           <Music className={`w-32 h-32 text-white/20 transition-transform duration-700 ${isPlaying ? 'scale-110' : 'scale-100'}`} />
        </div>

        {/* Bottom controls */}
        <div className="w-full bg-white/5 p-6">
          {/* Progress bar */}
          <div className="mb-6 flex items-center gap-4">
            <span className="text-xs font-medium text-white/70 w-10 text-right">
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
                className="h-12 w-12 rounded-full bg-white/10 text-white hover:bg-white/20 hover:text-white"
                onClick={togglePlay}
              >
                {isPlaying ? (
                  <Pause className="h-6 w-6 fill-current" />
                ) : (
                  <Play className="h-6 w-6 fill-current ml-1" />
                )}
              </Button>

              <div className="flex items-center gap-2 group">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-10 w-10 text-white/70 hover:bg-white/10 hover:text-white rounded-full"
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
                    className="h-8 px-3 text-sm font-medium text-white/70 hover:bg-white/10 hover:text-white rounded-full"
                  >
                    {playbackRate}x
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-32 bg-zinc-900 text-white border-white/10 z-[150]">
                  {[0.5, 0.75, 1, 1.25, 1.5, 2].map((speed) => (
                    <DropdownMenuItem
                      key={speed}
                      className={`cursor-pointer hover:bg-white/20 focus:bg-white/20 focus:text-white ${
                        playbackRate === speed ? "bg-white/10 font-bold" : ""
                      }`}
                      onClick={() => { handleSpeedChange(speed); }}
                    >
                      {speed}x
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default MusicPreviewModal;