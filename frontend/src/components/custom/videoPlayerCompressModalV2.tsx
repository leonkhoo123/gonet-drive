import { useState, useRef, useEffect, useCallback } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useDialogHistory } from "@/hooks/useDialogHistory";
import { useForceDarkStatusBar } from "@/hooks/useForceDarkStatusBar";
import { useVideoEventMetadata } from "@/hooks/useVideoPlayerV2/useVideoEventMetadata";
import { useVideoPlaybackController } from "@/hooks/useVideoPlayerV2/useVideoPlaybackController";
import { useVideoQualityPreference } from "@/hooks/useVideoPlayerV2/useVideoQualityPreference";
import { useLongPressSpeed } from "@/hooks/useVideoPlayerV2/useLongPressSpeed";
import { useVideoControlsVisibility } from "@/hooks/useVideoPlayerV2/useVideoControlsVisibility";
import { useVideoRename } from "@/hooks/useVideoPlayerV2/useVideoRename";
import { useVideoKeyboard } from "@/hooks/useVideoPlayerV2/useVideoKeyboard";
import { useVideoAutoPlay } from "@/hooks/useVideoPlayerV2/useVideoAutoPlay";
import type { FileInterface } from "@/api/api-file";
import type { VideoPlayerModalProps } from "./videoPlayerV2/types";
import { VideoSurface } from "./videoPlayerV2/VideoSurface";
import { VideoTimeDisplay } from "./videoPlayerV2/VideoTimeDisplay";
import { VideoProgressBar } from "./videoPlayerV2/VideoProgressBar";
import { VideoControls } from "./videoPlayerV2/VideoControls";
import { VideoRenameDialog } from "./videoPlayerV2/VideoRenameDialog";

/**
 * POC compressed video player.
 *
 * Same visual shell as VideoPlayerModalV2, but the quality selector routes
 * playback through a unified controller (useVideoPlaybackController):
 * `original` streams the raw file with native seeking, while 480/720/1080 use
 * the on-the-fly transcode endpoint and a JS virtual timeline.
 */
const VideoPlayerCompressModalV2 = ({
  file,
  isOpen,
  onClose,
  videoFiles = [],
  onSelectVideo,
}: VideoPlayerModalProps) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  // Start at the quality chosen earlier in this tab/PWA session; falls back to
  // original on a fresh session. The user drops to 480/720/1080 if bandwidth bites.
  const { quality, selectQuality, forceQuality } = useVideoQualityPreference();

  /* -------------------- playback (native or virtual timeline) -------------------- */
  const {
    isPlaying,
    isBuffering,
    playbackRate,
    progress,
    bufferedProgress,
    duration,
    currentTime,
    error,
    compressedUnavailable,
    togglePlay,
    skip,
    changeSpeed,
    commitSeek,
    scrubStart,
    scrubEnd,
    scrubTo,
  } = useVideoPlaybackController(videoRef, file.url, file.path, isOpen, quality);

  /* -------------------- event metadata -------------------- */
  const events = useVideoEventMetadata(isOpen, file.path, file.name);

  /* -------------------- control overlay -------------------- */
  const {
    showControls,
    setShowControls,
    clearHideTimer,
    startHideTimer,
    handlePressStart,
    handlePressEnd,
    handleHoverStart,
    handleHoverEnd,
  } = useVideoControlsVisibility(isOpen, isPlaying);

  /* -------------------- rename / flags -------------------- */
  const {
    newName,
    isNewName,
    disqualified,
    showRenameModal,
    tempName,
    setTempName,
    handleRenameSave,
    handleRenameDefault,
    handleRenameCancel,
    openRenameModal,
    handleDisqualified,
    resetRename,
  } = useVideoRename(file.name);

  /* -------------------- rotation -------------------- */
  const [rotation, setRotation] = useState(0);
  const [isRotation, setisRotation] = useState(false);

  /* -------------------- auto-play -------------------- */
  const handleSelectVideo = useCallback(
    (next: FileInterface) => {
      onSelectVideo?.(next);
    },
    [onSelectVideo]
  );

  const hasUnsavedMarks =
    disqualified || isNewName || rotation !== 0 || showRenameModal;

  const {
    mode: autoPlayMode,
    cycleMode: cycleAutoPlayMode,
    hasNext,
    shuffleRemaining,
  } = useVideoAutoPlay({
    isOpen,
    videoRef,
    filePath: file.path,
    videoFiles,
    onSelectVideo: handleSelectVideo,
    holdAdvance: hasUnsavedMarks,
  });

  // Switching clips (auto-play) must start the next one from a clean slate.
  useEffect(() => {
    setRotation(0);
    setisRotation(false);
    resetRename();
  }, [file.path, resetRename]);

  useForceDarkStatusBar(isOpen);

  /* -------------------- graceful fallback to original -------------------- */
  /* The compressed stream could not start: either the server is already
   * transcoding at capacity (busy) or ffmpeg failed. Original playback uses the
   * Range endpoint and needs no ffmpeg, so degrade instead of showing a dead
   * player. This is an error fallback, not a user choice, so it deliberately
   * does not overwrite the remembered quality: the next clip retries it.
   */
  useEffect(() => {
    if (!compressedUnavailable || quality === "original") return;
    toast.warning(
      compressedUnavailable === "busy"
        ? "Transcoding server is busy — switched to Original quality."
        : "Compressed playback unavailable — switched to Original quality."
    );
    forceQuality("original");
  }, [compressedUnavailable, quality, forceQuality]);

  /* -------------------- press and hold for 2x speed -------------------- */
  const {
    isBurst,
    wasLongPress,
    handleTouchStart,
    handleTouchMove,
    handleTouchEnd,
  } = useLongPressSpeed({ videoRef, baseRate: playbackRate });

  /* =====================================================
     VIDEO TAP BEHAVIOR
     ===================================================== */

  const handleVideoTap = () => {
    if (wasLongPress.current) return;

    if (showControls) {
      clearHideTimer();
      setShowControls(false);
    } else {
      setShowControls(true);
      startHideTimer();
    }
  };

  /* =====================================================
     ACTIONS
     ===================================================== */

  /** Jump to the next detected event start after the playhead. */
  const nextEvent = useCallback(() => {
    const next = events.find(([start]) => start > currentTime + 0.05);
    if (next) commitSeek(next[0]);
  }, [events, currentTime, commitSeek]);

  /** Jump to the previous detected event start before the playhead. */
  const prevEvent = useCallback(() => {
    let candidate: number | null = null;
    for (const [start] of events) {
      if (start < currentTime - 0.05) candidate = start;
      else break;
    }
    if (candidate !== null) commitSeek(candidate);
  }, [events, currentTime, commitSeek]);

  const handleRotation = () => {
    const currRotation = (rotation + 90) % 360;
    setRotation(currRotation);
    setisRotation(true);
    if (currRotation === 0) {
      setisRotation(false);
    }
  };

  /* =====================================================
     BACK BUTTON
     ===================================================== */
  const handleDismiss = useCallback((): void => {
    if (showRenameModal) {
      handleRenameCancel();
    } else {
      onClose(false, file.path, false, newName, 0);
    }
  }, [showRenameModal, file.path, newName, handleRenameCancel, onClose]);

  useDialogHistory(isOpen, handleDismiss);

  /* =====================================================
     KEYBOARD LISTENER
     ===================================================== */

  useVideoKeyboard({
    showRenameModal,
    onTogglePlay: togglePlay,
    onSkip: skip,
    onNextEvent: nextEvent,
    onPrevEvent: prevEvent,
    onRenameSave: handleRenameSave,
  });

  /* =====================================================
     RENDER
     ===================================================== */

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-black z-[100] select-none [-webkit-touch-callout:none]"
      onContextMenu={(e) => {
        // Disable the native menu for desktop right-click and mobile long press.
        e.preventDefault();
      }}
    >
      {/* VIDEO AREA */}
      <VideoSurface
        videoRef={videoRef}
        rotation={rotation}
        onTap={handleVideoTap}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      />

      {/* 2x SPEED BURST INDICATOR (while press-and-hold is active) */}
      {isBurst && (
        <div className="absolute top-12 left-1/2 -translate-x-1/2 z-20 pointer-events-none rounded-full bg-black/60 px-4 py-2 text-lg font-bold tracking-wider text-white">
          2× ▶▶
        </div>
      )}

      {/* COMPRESSION STATUS */}
      <div className="absolute top-3 left-3 z-20 pointer-events-none">
        <span className="rounded bg-black/50 px-2 py-1 text-xs font-semibold text-white/80">
          {quality === "original" ? "Original" : `${String(quality)}p`} ·{" "}
          {quality === "original" ? "Direct" : "Compressed"}
        </span>
      </div>

      {error && (
        <div className="absolute top-3 left-1/2 -translate-x-1/2 z-20 rounded bg-black/70 px-3 py-1 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* LOADING SPINNER — shown while a quality switch / seek reloads */}
      {isBuffering && !error && (
        <div className="absolute inset-0 z-20 flex items-center justify-center pointer-events-none">
          <Loader2 className="h-10 w-10 animate-spin text-white/80" />
        </div>
      )}

      {/* BOTTOM BACKGROUND FOR VISIBILITY */}
      <div
        className={`absolute bottom-0 left-0 w-full transition-all duration-300 pointer-events-none bg-black/60 ${
          showControls ? "h-12 opacity-100" : "h-0 opacity-0"
        }`}
      />

      {/* --- Time Display --- */}
      <VideoTimeDisplay
        currentTime={currentTime}
        duration={duration}
        disqualified={disqualified}
        isNewName={isNewName}
        newName={newName}
        fileName={file.name}
        isRotation={isRotation}
        rotation={rotation}
      />

      {/* PROGRESS */}
      <VideoProgressBar
        showControls={showControls}
        bufferedProgress={bufferedProgress}
        progress={progress}
        duration={duration}
        currentTime={currentTime}
        events={events}
        onScrubStart={scrubStart}
        onScrubEnd={scrubEnd}
        onScrub={scrubTo}
        onHoverStart={handleHoverStart}
        onHoverEnd={handleHoverEnd}
      />

      {/* CONTROLS */}
      <VideoControls
        showControls={showControls}
        isPlaying={isPlaying}
        playbackRate={playbackRate}
        hasEvents={events.length > 0}
        autoPlayMode={autoPlayMode}
        hasNext={hasNext}
        shuffleRemaining={shuffleRemaining}
        onPressStart={handlePressStart}
        onPressEnd={handlePressEnd}
        onHoverStart={handleHoverStart}
        onHoverEnd={handleHoverEnd}
        onSkip={skip}
        onPrevEvent={prevEvent}
        onNextEvent={nextEvent}
        onTogglePlay={togglePlay}
        onCycleAutoPlayMode={cycleAutoPlayMode}
        onChangeSpeed={changeSpeed}
        onOpenRename={openRenameModal}
        onToggleDisqualified={handleDisqualified}
        onRotate={handleRotation}
        onClose={() => {
          onClose(disqualified, file.path, isNewName, newName, rotation);
        }}
        quality={quality}
        onChangeQuality={selectQuality}
      />

      {/* RENAME MODAL */}
      {showRenameModal && (
        <VideoRenameDialog
          tempName={tempName}
          onChangeName={setTempName}
          onDefault={handleRenameDefault}
          onCancel={handleRenameCancel}
          onSave={handleRenameSave}
        />
      )}
    </div>
  );
};

export default VideoPlayerCompressModalV2;
