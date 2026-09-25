import { useState, useRef, useEffect, useCallback } from "react";
import { useDialogHistory } from "@/hooks/useDialogHistory";
import { useForceDarkStatusBar } from "@/hooks/useForceDarkStatusBar";
import { useVideoEventMetadata } from "@/hooks/useVideoPlayerV2/useVideoEventMetadata";
import { useVideoPlayback } from "@/hooks/useVideoPlayerV2/useVideoPlayback";
import { useVideoSwipeSeek } from "@/hooks/useVideoPlayerV2/useVideoSwipeSeek";
import { useVideoControlsVisibility } from "@/hooks/useVideoPlayerV2/useVideoControlsVisibility";
import { useVideoRename } from "@/hooks/useVideoPlayerV2/useVideoRename";
import { useVideoKeyboard } from "@/hooks/useVideoPlayerV2/useVideoKeyboard";
import type { VideoPlayerModalProps } from "./videoPlayerV2/types";
import { VideoSurface } from "./videoPlayerV2/VideoSurface";
import { VideoDragDelta } from "./videoPlayerV2/VideoDragDelta";
import { VideoTimeDisplay } from "./videoPlayerV2/VideoTimeDisplay";
import { VideoProgressBar } from "./videoPlayerV2/VideoProgressBar";
import { VideoControls } from "./videoPlayerV2/VideoControls";
import { VideoRenameDialog } from "./videoPlayerV2/VideoRenameDialog";

const VideoPlayerModalV2 = ({
  file,
  isOpen,
  onClose,
}: VideoPlayerModalProps) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  /* -------------------- playback -------------------- */
  const {
    isPlaying,
    playbackRate,
    progress,
    bufferedProgress,
    duration,
    currentTime,
    wasPlayingBeforeScrub,
    safePlay,
    togglePlay,
    skip,
    changeSpeed,
    seekTo,
    scrubStart,
    scrubEnd,
    scrubTo,
  } = useVideoPlayback(videoRef);

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
  } = useVideoRename(file.name);

  /* -------------------- rotation -------------------- */
  const [rotation, setRotation] = useState(0);
  const [isRotation, setisRotation] = useState(false);

  useForceDarkStatusBar(isOpen);

  /* -------------------- swipe to seek -------------------- */
  const {
    dragDeltaSeconds,
    hasScrubbed,
    handleTouchStart,
    handleTouchMove,
    handleTouchEnd,
  } = useVideoSwipeSeek({
    videoRef,
    duration,
    wasPlayingBeforeScrub,
    onSeek: seekTo,
    onResume: scrubEnd,
  });

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

  /* =====================================================
     VIDEO LOAD
     ===================================================== */

  useEffect(() => {
    if (!isOpen) return;

    const timer = setTimeout(() => {
      const video = videoRef.current;
      if (!video) return;

      video.src = file.url;
      safePlay();
    }, 100);

    return () => {
      clearTimeout(timer);
    };
  }, [isOpen, file.url, safePlay]);

  /* =====================================================
     ACTIONS
     ===================================================== */

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

      {/* DRAG DELTA POPUP */}
      {dragDeltaSeconds !== null && (
        <VideoDragDelta seconds={dragDeltaSeconds} />
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
      {/* z-30 keeps the seek bar above the time display (z-10) and the
          controls column, so the full width — including the far left and
          the area under the buttons — stays clickable for scrubbing. */}
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
        onPressStart={handlePressStart}
        onPressEnd={handlePressEnd}
        onHoverStart={handleHoverStart}
        onHoverEnd={handleHoverEnd}
        onSkip={skip}
        onPrevEvent={prevEvent}
        onNextEvent={nextEvent}
        onTogglePlay={togglePlay}
        onChangeSpeed={changeSpeed}
        onOpenRename={openRenameModal}
        onToggleDisqualified={handleDisqualified}
        onRotate={handleRotation}
        onClose={() => {
          onClose(disqualified, file.path, isNewName, newName, rotation);
        }}
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

export default VideoPlayerModalV2;
