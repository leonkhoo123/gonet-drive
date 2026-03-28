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
} from "lucide-react";
import type { FileInterface } from "@/api/api-file";

import { useDialogHistory } from "@/hooks/useDialogHistory";

interface VideoPlayerCompressModalProps {
  file: FileInterface;
  isOpen: boolean;
  onClose: (
    isDisqualified: boolean,
    oriPath: string,
    isNewName: boolean,
    newName: string
  ) => void;
}

const VideoPlayerCompressModal: React.FC<VideoPlayerCompressModalProps> = ({
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
  const [bufferedProgress, setBufferedProgress] = useState(0); // %
  const [error, setError] = useState<string | null>(null);

  const formatTime = (time: number): string => {
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes}:${seconds.toString().padStart(2, "0")}`;
  };

  // Load compressed stream
  useEffect(() => {
    if (!isOpen) return;
    const video = videoRef.current;
    if (!video) return;

    const timer = setTimeout(() => {
      try {
        // Construct compress endpoint by replacing /play/file with /play/compress/file in the file.url
        const compressedUrl = file.url.replace('/api/user/video/play/file', '/api/user/video/play/compress/file');
        console.log("compressurl",compressedUrl);
        video.src = compressedUrl;
        // eslint-disable-next-line @typescript-eslint/use-unknown-in-catch-callback-variable
        video.play().catch((err) => {
          setIsPlaying(false);
          console.error("Playback start error:", err);
        });
        setError(null);
      } catch (err) {
        console.error("Video load failed:", err);
        setError("Unable to load compressed stream.");
      }
    }, 100);

    return () => { clearTimeout(timer); };
  }, [isOpen, file.path]);

  // Attach video event listeners
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
    const handleProgress = () => {
      if (!video.duration || video.buffered.length === 0) return;
      const lastBufferedTime = video.buffered.end(video.buffered.length - 1);
      const bufferedPercent = (lastBufferedTime / video.duration) * 100;
      setBufferedProgress(bufferedPercent);
    };

    video.addEventListener("play", handlePlay);
    video.addEventListener("pause", handlePause);
    video.addEventListener("loadedmetadata", handleLoadedMetadata);
    video.addEventListener("timeupdate", handleTimeUpdate);
    video.addEventListener("progress", handleProgress);

    return () => {
      video.removeEventListener("play", handlePlay);
      video.removeEventListener("pause", handlePause);
      video.removeEventListener("loadedmetadata", handleLoadedMetadata);
      video.removeEventListener("timeupdate", handleTimeUpdate);
      video.removeEventListener("progress", handleProgress);
    };
  }, []);

  // --- Controls ---
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

  // --- Rename Modal Logic ---
  const openRenameModal = () => {
    setTempName(newName);
    setShowRenameModal(true);
  };

  const handleRenameCancel = useCallback(() => { setShowRenameModal(false); }, []);

  const handleRenameSave = () => {
    let finalName = tempName.trim();
    const originalExt = file.name.includes(".")
      ? file.name.substring(file.name.lastIndexOf("."))
      : "";

    if (!finalName.includes(".") && originalExt) {
      finalName += originalExt;
    }

    if (finalName !== file.name) {
      setNewname(finalName);
      setisNewname(true);
    }
    setShowRenameModal(false);
  };

  const handleRenameDefault = () => {
    setNewname("");
    setisNewname(false);
    setShowRenameModal(false);
  };

  const handleDismiss = useCallback(() => {
    if (showRenameModal) {
      handleRenameCancel();
    } else {
      onClose(disqualified, file.path, isNewName, newName);
    }
  }, [showRenameModal, disqualified, file.path, isNewName, newName, onClose, handleRenameCancel]);

  useDialogHistory(isOpen, handleDismiss);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-10 bg-black">
      <div className="flex items-center justify-center bg-black h-full">
        {/* video element */}
        <video
          preload="auto"
          ref={videoRef}
          controls={false}
          autoPlay
          className="max-h-[calc(100dvh)] max-w-full object-contain"
        />
      </div>

      {/* Error message */}
      {error && (
        <div className="absolute top-5 left-5 text-red-400 text-sm bg-black/50 px-2 py-1 rounded">
          {error}
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
        <span className="ml-3 text-primary font-bold">Compressed</span>

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
      </div>

      {/* --- Progress Bar ---*/}
      <div
        className="absolute bottom-0 w-full h-2 bg-gray-700 cursor-pointer"
        onClick={handleSeek}
      >
        {/* Buffered */}
        <div
          className="h-2 bg-white/20 absolute top-0 left-0 transition-all"
          style={{ width: `${bufferedProgress}%` }}
        />
        {/* Played */}
        <div
          className="h-2 bg-primary transition-all absolute top-0 left-0"
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* --- Control Bar --- */}
      <div className="absolute bottom-1/8 right-0 p-2 bg-transparent rounded-md flex flex-col items-center justify-center sm:w-[100px] w-[70px] space-y-2">
        <Button
          variant="ghost"
          onClick={() => { skip(-1); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <SkipBack className="h-4 w-4 mr-1" /> 1s
        </Button>

        <Button
          variant="ghost"
          onClick={() => { skip(3); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        <Button
          variant="ghost"
          onClick={() => { skip(1); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 2.0); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x2"}
        </Button>

        <Button
          variant="ghost"
          onClick={() => { changeSpeed(playbackRate !== 1.0 ? 1.0 : 3.0); }}
          className="hover:bg-white/80 w-full bg-white/30"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x3"}
        </Button>

        <div className="text-sm text-white">{playbackRate.toFixed(1)}x</div>

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

        <Button
          variant="ghost"
          size="icon"
          onClick={openRenameModal}
          className="hover:bg-green-300/80 w-full bg-green-300/30 mt-5"
        >
          <TextCursorInput className="h-5 w-5" />
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={handleDisqualified}
          className="hover:bg-red-300/80 w-full bg-red-300/30 mt-5"
        >
          <ListX className="h-5 w-5" />
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={handleDismiss}
          className="hover:bg-white/80 w-full bg-white/30 mt-5"
        >
          <LogOut className="h-5 w-5" />
        </Button>
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

export default VideoPlayerCompressModal;
