import { useEffect, useRef, useState, useCallback } from "react";
import { X, Play, Pause, Volume2, VolumeX, SkipBack, SkipForward, Headphones } from "lucide-react";
import { type FileInterface } from "@/api/api-file";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { toast } from "sonner";
import { formatTime } from "@/lib/utils";
import TrackInfo from "./trackInfo";

interface MusicPlayerProps {
  file: FileInterface;
  playlist?: FileInterface[];
  onSelectMusic?: (file: FileInterface) => void;
  onClose: () => void;
  forcePause?: boolean;
}



export function MusicPlayer({ file, playlist = [], onSelectMusic, onClose, forcePause }: MusicPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [allowBackground, setAllowBackground] = useState(true);
  const [activePlaylist, setActivePlaylist] = useState<FileInterface[]>(playlist);

  // Pause music if forcePause becomes true
  useEffect(() => {
    if (forcePause) {
      setIsPlaying(false);
    }
  }, [forcePause]);

  // Update active playlist only when the currently playing file is in the viewed folder's playlist
  useEffect(() => {
    const isFileInPlaylist = playlist.some((p) => p.url === file.url);
    if (isFileInPlaylist) {
      setActivePlaylist(playlist);
    }
  }, [file.url, playlist]);

  // Play/pause logic
  useEffect(() => {
    if (audioRef.current) {
      if (isPlaying) {
        audioRef.current.play().catch((e: unknown) => { console.error("Error playing audio:", e); });
      } else {
        audioRef.current.pause();
      }
    }
  }, [isPlaying, file]);

  // Handle visibility change to stop audio if background playing is disabled
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden && !allowBackground && audioRef.current && !audioRef.current.paused) {
        audioRef.current.pause();
        setIsPlaying(false);
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [allowBackground]);

  const handleNext = useCallback(() => {
    if (activePlaylist.length > 0 && onSelectMusic) {
      const currentIndex = activePlaylist.findIndex((p) => p.name === file.name);
      if (currentIndex !== -1) {
        const nextIndex = (currentIndex + 1) % activePlaylist.length;
        onSelectMusic(activePlaylist[nextIndex]);
      }
    }
  }, [file.name, onSelectMusic, activePlaylist]);

  const handlePrev = useCallback(() => {
    if (activePlaylist.length > 0 && onSelectMusic) {
      const currentIndex = activePlaylist.findIndex((p) => p.name === file.name);
      if (currentIndex !== -1) {
        const prevIndex = (currentIndex - 1 + activePlaylist.length) % activePlaylist.length;
        onSelectMusic(activePlaylist[prevIndex]);
      }
    }
  }, [file.name, onSelectMusic, activePlaylist]);

  // Set media session metadata for lock screen integration
  useEffect(() => {
    if ('mediaSession' in navigator) {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: file.name,
        artist: 'Cloud Drive Music',
      });

      navigator.mediaSession.setActionHandler('play', () => { setIsPlaying(true); });
      navigator.mediaSession.setActionHandler('pause', () => { setIsPlaying(false); });
      if (activePlaylist.length > 1 && onSelectMusic) {
        navigator.mediaSession.setActionHandler('previoustrack', handlePrev);
        navigator.mediaSession.setActionHandler('nexttrack', handleNext);
      } else {
        navigator.mediaSession.setActionHandler('previoustrack', null);
        navigator.mediaSession.setActionHandler('nexttrack', null);
      }
    }
  }, [file.name, activePlaylist, onSelectMusic, handleNext, handlePrev]);

  // Auto-play when file changes
  useEffect(() => {
    setIsPlaying(true);
    setCurrentTime(0);
  }, [file.url]);

  const togglePlay = () => { setIsPlaying(!isPlaying); };

  const handleTimeUpdate = () => {
    if (audioRef.current) {
      setCurrentTime(audioRef.current.currentTime);
    }
  };

  const handleLoadedMetadata = () => {
    if (audioRef.current) {
      setDuration(audioRef.current.duration);
    }
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
    if (audioRef.current) {
      audioRef.current.volume = newVolume;
    }
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
    if (activePlaylist.length > 0 && onSelectMusic) {
      handleNext();
    } else {
      setIsPlaying(false);
      setCurrentTime(0);
    }
  };

  return (
    <div className="w-full bg-background border-t border-border shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] p-3 md:px-6 flex flex-col shrink-0 z-50 relative">
      <div className="absolute top-0 left-0 right-0 -translate-y-1/2 px-0 z-10">
        <Slider
          value={[currentTime]}
          max={duration || 100}
          step={1}
          onValueChange={handleSeek}
          className="w-full h-1"
        />
      </div>

      <audio
        ref={audioRef}
        src={file.url}
        onTimeUpdate={handleTimeUpdate}
        onLoadedMetadata={handleLoadedMetadata}
        onEnded={handleEnded}
        className="hidden"
      />

      <div className="flex flex-row items-center justify-between gap-2 md:gap-4 w-full">
        {/* Left: Controls and Time */}
        <div className="flex items-center gap-1 md:gap-4 md:w-[30%] shrink-0">
          <div className="flex items-center gap-1 md:gap-2">
            <Button
              variant="ghost"
              size="icon"
              className="w-8 h-8 rounded-full"
              onClick={handlePrev}
              disabled={activePlaylist.length <= 1}
            >
              <SkipBack className="w-4 h-4" />
            </Button>
            <Button
              variant="secondary"
              size="icon"
              className="w-10 h-10 rounded-full shrink-0"
              onClick={togglePlay}
            >
              {isPlaying ? <Pause className="w-5 h-5" /> : <Play className="w-5 h-5 ml-1" />}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="w-8 h-8 rounded-full"
              onClick={handleNext}
              disabled={activePlaylist.length <= 1}
            >
              <SkipForward className="w-4 h-4" />
            </Button>
          </div>
          <span className="text-xs text-muted-foreground hidden md:block whitespace-nowrap">
            {formatTime(currentTime)} / {formatTime(duration)}
          </span>
        </div>

        {/* Middle: Track Info */}
        <TrackInfo text={file.name} currentTime={currentTime} duration={duration} />

        {/* Right: Extra Controls */}
        <div className="flex items-center justify-end gap-1 md:gap-4 md:w-[30%] shrink-0">
          <div className="hidden md:flex items-center gap-2 bg-muted/50 p-1.5 rounded-md">
            <Switch
              id="bg-play"
              checked={allowBackground}
              onCheckedChange={(checked) => {
                setAllowBackground(checked);
                toast.success(`Background Play ${checked ? "Enabled" : "Disabled"}`);
              }}
              className="scale-75 data-[state=checked]:bg-primary"
            />
            <Label htmlFor="bg-play" className="text-[10px] font-medium leading-none cursor-pointer">
              BG Play
            </Label>
          </div>

          <div className="hidden md:flex items-center gap-2 w-28">
            <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={toggleMute}>
              {isMuted || volume === 0 ? <VolumeX className="w-4 h-4" /> : <Volume2 className="w-4 h-4" />}
            </Button>
            <Slider
              value={[isMuted ? 0 : volume]}
              max={1}
              step={0.01}
              onValueChange={handleVolumeChange}
              className="w-full"
            />
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
            <Popover>
              <PopoverTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0">
                  {isMuted || volume === 0 ? <VolumeX className="w-4 h-4" /> : <Volume2 className="w-4 h-4" />}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-12 h-32 p-2 mb-2 bg-background flex justify-center" side="top" align="center">
                <Slider
                  orientation="vertical"
                  value={[isMuted ? 0 : volume]}
                  max={1}
                  step={0.01}
                  onValueChange={handleVolumeChange}
                  className="h-full"
                />
              </PopoverContent>
            </Popover>
          </div>

          <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0 md:ml-2 hover:bg-destructive/10 hover:text-destructive" onClick={onClose}>
            <X className="w-5 h-5" />
          </Button>
        </div>
      </div>
    </div>
  );
}
