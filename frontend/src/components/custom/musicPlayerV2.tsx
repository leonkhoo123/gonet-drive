import { useEffect, useRef, useState, useCallback } from "react";
import { X, Play, Pause, Volume2, VolumeX, SkipBack, SkipForward, Headphones, ListMusic } from "lucide-react";
import { type FileInterface } from "@/api/api-file";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { toast } from "sonner";
import TrackInfo from "./trackInfo";
import { MusicQueueModal } from "./MusicQueueModal";

interface MusicPlayerProps {
  file: FileInterface | null;
  playlist?: FileInterface[];
  onSelectMusic?: (file: FileInterface) => void;
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

export function MusicPlayerV2({ file, playlist = [], onSelectMusic, onClose, forcePause }: MusicPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const playerContainerRef = useRef<HTMLDivElement>(null);
  const isInternalChange = useRef(false);
  const [currentFile, setCurrentFile] = useState<FileInterface | null>(file);
  const [isVisible, setIsVisible] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [allowBackground, setAllowBackground] = useState(true);
  const [activePlaylist, setActivePlaylist] = useState<FileInterface[]>(playlist);
  const [isVolumeOpen, setIsVolumeOpen] = useState(false);
  const [isQueueOpen, setIsQueueOpen] = useState(false);
  const [playerHeight, setPlayerHeight] = useState(0);

  // Update player height for queue modal positioning
  useEffect(() => {
    if (!playerContainerRef.current) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setPlayerHeight(entry.target.clientHeight);
      }
    });
    observer.observe(playerContainerRef.current);
    return () => { observer.disconnect(); };
  }, [isVisible]);

  useEffect(() => {
    if (file) {
      if (!currentFile) {
        // First time opening: set file, then trigger animation after render
        setCurrentFile(file);
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            setIsVisible(true);
          });
        });
      } else {
        // Just switching songs or restarting the same song
        if (currentFile.url === file.url) {
          setIsPlaying(true);
          setCurrentTime(0);
          if (audioRef.current) {
            audioRef.current.currentTime = 0;
          }
        }
        setCurrentFile(file);
        setIsVisible(true);
      }
    } else {
      setIsVisible(false);
      setIsPlaying(false);
      if (audioRef.current) {
        audioRef.current.pause();
      }
    }
  }, [file]);

  // Pause music if forcePause becomes true
  useEffect(() => {
    if (forcePause) {
      setIsPlaying(false);
    }
  }, [forcePause]);

  const [lastPlaylistKey, setLastPlaylistKey] = useState<string>("");
  const [lastFileUrl, setLastFileUrl] = useState<string>("");

  // Update active playlist only when the currently playing file is in the viewed folder's playlist
  // Reorder it to put the newly selected file at the top, UNLESS the selection came from within the player (Next/Prev/Queue)
  useEffect(() => {
    if (!currentFile) return;
    const isFileInPlaylist = playlist.some((p) => p.url === currentFile.url);
    const newPlaylistKey = playlist.map(p => p.url).join(',');
    
    if (isFileInPlaylist) {
      if (!isInternalChange.current && (currentFile.url !== lastFileUrl || newPlaylistKey !== lastPlaylistKey)) {
        // User clicked from outside the player (e.g., file list) or folder changed
        // Keep the playlist as is so the queue order remains exactly the same as the folder.
        // We will just scroll to the active item in the queue.
        setActivePlaylist([...playlist]);
        setLastPlaylistKey(newPlaylistKey);
        setLastFileUrl(currentFile.url);
      } else if (isInternalChange.current) {
        // Just reset the flag, don't override user's queue or shuffle
        isInternalChange.current = false;
        setLastFileUrl(currentFile.url);
        setLastPlaylistKey(newPlaylistKey);
        // We can just keep the current activePlaylist since they clicked inside the queue
      } else if (newPlaylistKey !== lastPlaylistKey) {
        // If the folder actually changed while we weren't internally changing tracks
        // (e.g., a file was added/removed, or we navigated to a new folder but currentFile didn't change)
        setLastPlaylistKey(newPlaylistKey);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentFile?.url, playlist]);

  // Play/pause logic
  useEffect(() => {
    if (audioRef.current) {
      if (isPlaying) {
        audioRef.current.play().catch((e: unknown) => { console.error("Error playing audio:", e); });
      } else {
        audioRef.current.pause();
      }
    }
  }, [isPlaying, currentFile]);

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
    if (activePlaylist.length > 0 && onSelectMusic && currentFile) {
      const currentIndex = activePlaylist.findIndex((p) => p.name === currentFile.name);
      if (currentIndex !== -1) {
        const nextIndex = (currentIndex + 1) % activePlaylist.length;
        isInternalChange.current = true;
        onSelectMusic(activePlaylist[nextIndex]);
      }
    }
  }, [currentFile?.name, onSelectMusic, activePlaylist]);

  const handlePrev = useCallback(() => {
    if (activePlaylist.length > 0 && onSelectMusic && currentFile) {
      const currentIndex = activePlaylist.findIndex((p) => p.name === currentFile.name);
      if (currentIndex !== -1) {
        const prevIndex = (currentIndex - 1 + activePlaylist.length) % activePlaylist.length;
        isInternalChange.current = true;
        onSelectMusic(activePlaylist[prevIndex]);
      }
    }
  }, [currentFile?.name, onSelectMusic, activePlaylist]);

  // Set media session metadata for lock screen integration
  useEffect(() => {
    if ('mediaSession' in navigator && currentFile) {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: currentFile.name,
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
  }, [currentFile?.name, activePlaylist, onSelectMusic, handleNext, handlePrev]);

  // Auto-play when file changes
  useEffect(() => {
    if (currentFile) {
      setIsPlaying(true);
      setCurrentTime(0);
    }
  }, [currentFile?.url]);

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

  if (!currentFile) return null;

  return (
    <div
      ref={playerContainerRef}
      className={`overflow-visible min-h-0 flex flex-col w-full bg-background border-border shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] shrink-0 z-50 transition-all duration-500 ease-in-out relative ${isVisible ? "grid-rows-[1fr] opacity-100 border-t" : "grid-rows-[0fr] opacity-0 border-t-0"
        }`}
    >
      {/* Music Queue Modal Container */}
      <div 
        className={`md:hidden fixed inset-x-0 top-0 z-40 bg-background transition-opacity duration-300 ease-in-out flex flex-col ${isQueueOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}
        style={{ bottom: playerHeight > 0 ? playerHeight : undefined }}
      >
        <MusicQueueModal
          isOpen={isQueueOpen}
          onClose={() => { setIsQueueOpen(false); }}
          playlist={activePlaylist}
          currentFile={currentFile}
          onSelectMusic={(f) => { 
            isInternalChange.current = true;
            if (onSelectMusic) onSelectMusic(f); 
            setIsQueueOpen(false); 
          }}
          onReorderPlaylist={setActivePlaylist}
        />
      </div>

      <div className={`w-full p-3 md:px-6 flex flex-col transition-all duration-700 ${isVisible ? 'visible opacity-100' : 'invisible opacity-0'}`}>
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
          src={currentFile.url}
          onTimeUpdate={handleTimeUpdate}
          onLoadedMetadata={handleLoadedMetadata}
          onEnded={handleEnded}
          className="hidden"
        />

        <div className="flex flex-row items-center justify-between gap-1 md:gap-4 w-full">
          {/* Left: Controls and Time */}
          <div className="flex items-center gap-2 md:gap-4 shrink-0 w-auto md:w-[30%]">
            <div className="flex items-center gap-0.5 md:gap-2">
              <Button
                variant="ghost"
                size="icon"
                className="w-8 h-8 rounded-full shrink-0"
                onClick={handlePrev}
                disabled={activePlaylist.length <= 1}
              >
                <SkipBack className="w-4 h-4" />
              </Button>
              <Button
                variant="secondary"
                size="icon"
                className="w-9 h-9 md:w-10 md:h-10 rounded-full shrink-0"
                onClick={togglePlay}
              >
                {isPlaying ? <Pause className="w-4 h-4 md:w-5 md:h-5" /> : <Play className="w-4 h-4 md:w-5 md:h-5 ml-1" />}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="w-8 h-8 rounded-full shrink-0"
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
          <div 
            className="flex-1 min-w-0 flex justify-center cursor-pointer md:cursor-default" 
            onClick={() => { 
              // Only open the full-screen mobile modal if on a small screen
              if (window.innerWidth < 768) {
                setIsQueueOpen(!isQueueOpen); 
              }
            }}
          >
            <TrackInfo text={currentFile.name} currentTime={currentTime} duration={duration} />
          </div>

          {/* Right: Extra Controls */}
          <div className="flex items-center justify-end gap-1 md:gap-4 shrink-0 w-auto md:w-[30%]">
            <div className="hidden md:flex items-center gap-2 bg-muted/50 p-1.5 rounded-md">
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 shrink-0 mr-1"
                    title="Music Queue"
                  >
                    <ListMusic className="w-4 h-4" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent 
                  className="w-96 p-0 mb-4 h-[500px] max-h-[70vh] flex flex-col shadow-xl border-border bg-background" 
                  side="top" 
                  align="center"
                  sideOffset={16}
                >
                  <MusicQueueModal
                    isOpen={true}
                    onClose={() => { /* no-op for popover */ }}
                    playlist={activePlaylist}
                    currentFile={currentFile}
                    onSelectMusic={(f) => { 
                      isInternalChange.current = true;
                      if (onSelectMusic) onSelectMusic(f); 
                    }}
                    onReorderPlaylist={setActivePlaylist}
                  />
                </PopoverContent>
              </Popover>
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
              <Popover open={isVolumeOpen} onOpenChange={setIsVolumeOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    onClick={(e) => {
                      if (isVolumeOpen) {
                        e.preventDefault();
                        toggleMute();
                      }
                    }}
                    onPointerDown={(e) => {
                      if (isVolumeOpen) {
                        e.preventDefault();
                      }
                    }}
                  >
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
    </div>
  );
}
