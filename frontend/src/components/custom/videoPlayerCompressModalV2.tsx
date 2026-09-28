import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useDialogHistory } from "@/hooks/useDialogHistory";
import { useForceDarkStatusBar } from "@/hooks/useForceDarkStatusBar";
import { useVideoEventEditor } from "@/hooks/useVideoPlayerV2/useVideoEventEditor";
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
import { EventEditorBar } from "./videoPlayerV2/EventEditorBar";
import { EventEditToolbar } from "./videoPlayerV2/EventEditToolbar";
import { EventInspector } from "./videoPlayerV2/EventInspector";
import { ConfirmDialog } from "./videoPlayerV2/ConfirmDialog";

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
  onVideoMutation,
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

  /* -------------------- highlights (markers + editor) -------------------- */
  const editor = useVideoEventEditor({
    isOpen,
    filePath: file.path,
    fileName: file.name,
    duration,
    currentTime,
  });
  const { save: saveEvents } = editor;

  // Pending action to run once the user confirms discarding unsaved highlights.
  const [discardAction, setDiscardAction] = useState<(() => void) | null>(null);
  // Pending terminal save: null when no Done confirmation is open. Captures the
  // rename/rotation at press time so the dialog copy stays stable.
  const [pendingCommit, setPendingCommit] = useState<{
    newName: string;
    rotation: number;
  } | null>(null);

  const requestDiscard = useCallback(
    (action: () => void) => {
      if (!editor.isDirty) {
        action();
        return;
      }
      setDiscardAction(() => action);
    },
    [editor.isDirty],
  );

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
  } = useVideoControlsVisibility(isOpen, isPlaying, editor.isEditing);

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
    disqualified ||
    isNewName ||
    rotation !== 0 ||
    showRenameModal ||
    editor.isEditing;

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
    // The editor owns the timeline; a tap must not hide the pinned controls.
    // It does, however, drop the current event selection (and hide the
    // inspector), matching the "tap empty space to deselect" convention.
    if (editor.isEditing) {
      editor.select(null);
      return;
    }

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

  // Navigation follows the live draft while editing so the Evt buttons land on
  // the events as you just edited them, not the saved/loaded positions.
  const navEvents = editor.isEditing ? editor.draft : editor.events;

  /** Jump to the next event start after the playhead. */
  const nextEvent = useCallback(() => {
    const next = navEvents.find(([start]) => start > currentTime + 0.05);
    if (next) commitSeek(next[0]);
  }, [navEvents, currentTime, commitSeek]);

  /** Jump to the previous event start before the playhead. */
  const prevEvent = useCallback(() => {
    let candidate: number | null = null;
    for (const [start] of navEvents) {
      if (start < currentTime - 0.05) candidate = start;
      else break;
    }
    if (candidate !== null) commitSeek(candidate);
  }, [navEvents, currentTime, commitSeek]);

  const handleRotation = () => {
    const currRotation = (rotation + 90) % 360;
    setRotation(currRotation);
    setisRotation(true);
    if (currRotation === 0) {
      setisRotation(false);
    }
  };

  /**
   * Done in edit mode is terminal: it always confirms, then saves and closes the
   * player so the browser drops its (now rewritten) media cache. The copy and
   * the commit differ when a rename is staged; rotation only rides along with a
   * rename (no rename, no rotate).
   */
  const requestCommit = useCallback(() => {
    setPendingCommit({
      newName: isNewName ? newName : "",
      rotation: isNewName ? rotation : 0,
    });
  }, [isNewName, newName, rotation]);

  const runCommit = useCallback(async () => {
    const action = pendingCommit;
    if (!action) return;
    setPendingCommit(null);
    const ok = await saveEvents({
      newName: action.newName || undefined,
      rotation: action.rotation,
    });
    if (!ok) return;
    // Close first so the player disappears immediately; the host refresh then
    // runs against the now-current folder (rename has already moved the file).
    onClose(false, file.path, false, "", 0);
    await onVideoMutation?.();
  }, [pendingCommit, saveEvents, onVideoMutation, onClose, file.path]);

  /* =====================================================
     BACK BUTTON
     ===================================================== */
  const handleDismiss = useCallback((): void => {
    if (showRenameModal) {
      handleRenameCancel();
      return;
    }
    // An open discard dialog swallows Back: it just cancels the dialog.
    if (discardAction) {
      setDiscardAction(null);
      return;
    }
    // An open Done confirmation swallows Back too.
    if (pendingCommit) {
      setPendingCommit(null);
      return;
    }
    // Back mirrors the editor's Cancel: discard the draft (with a confirm when
    // it differs from the saved highlights) rather than closing the player.
    if (editor.isEditing) {
      requestDiscard(editor.cancel);
      return;
    }
    onClose(false, file.path, false, newName, 0);
  }, [
    showRenameModal,
    discardAction,
    pendingCommit,
    editor,
    file.path,
    newName,
    handleRenameCancel,
    requestDiscard,
    onClose,
  ]);

  useDialogHistory(isOpen, handleDismiss);

  /* =====================================================
     KEYBOARD LISTENER
     ===================================================== */

  // Destructured so the memo deps are the stable callbacks, not the whole
  // (per-render) editor object.
  const {
    isEditing: isEditingHighlights,
    setEdgeToPlayhead,
    deleteSelected,
  } = editor;

  // Stable identity so the document keydown listener is not re-bound on every
  // playhead update.
  const eventEditorShortcuts = useMemo(
    () =>
      isEditingHighlights && !discardAction
        ? {
            onSetIn: () => { setEdgeToPlayhead("start"); },
            onSetOut: () => { setEdgeToPlayhead("end"); },
            onDelete: deleteSelected,
          }
        : undefined,
    [isEditingHighlights, discardAction, setEdgeToPlayhead, deleteSelected]
  );

  useVideoKeyboard({
    showRenameModal,
    onTogglePlay: togglePlay,
    onSkip: skip,
    onNextEvent: nextEvent,
    onPrevEvent: prevEvent,
    onRenameSave: handleRenameSave,
    onEscape: handleDismiss,
    // `<`/`,` and `>`/`.` step the playback speed in the compress player.
    playbackRate,
    onChangeSpeed: changeSpeed,
    eventEditor: eventEditorShortcuts,
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

      {/* CUSTOM SPEED INDICATOR (persistent while the rate is not 1×) */}
      {!isBurst && playbackRate !== 1 && (
        <div className="absolute top-12 left-1/2 -translate-x-1/2 z-20 pointer-events-none rounded-full bg-black/60 px-4 py-2 text-lg font-bold tracking-wider text-white">
          {playbackRate}× {playbackRate > 1 ? "▶▶" : "▶"}
        </div>
      )}

      {/* COMPRESSION STATUS + EVENT METADATA SOURCE */}
      <div className="absolute top-3 left-3 z-20 pointer-events-none flex items-center gap-1">
        <span className="rounded bg-black/50 px-2 py-1 text-xs font-semibold text-white/80">
          {quality === "original" ? "Original" : `${String(quality)}p`} ·{" "}
          {quality === "original" ? "Direct" : "Compressed"}
        </span>
        {editor.metadataSource && (
          <span className="rounded bg-black/50 px-2 py-1 text-xs font-semibold text-white/80">
            Events: {editor.metadataSource === "embedded" ? "embed" : "sidecar"}
          </span>
        )}
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

      {/* TIMELINE — the editor bar replaces the scrub bar while editing */}
      {editor.isEditing ? (
        <EventEditorBar
          events={editor.draft}
          duration={duration}
          currentTime={currentTime}
          selectedIndex={editor.selectedIndex}
          onSelect={editor.select}
          onSeek={commitSeek}
          onBeginChange={editor.beginChange}
          onPreviewChange={editor.previewChange}
          onEndChange={editor.endChange}
        />
      ) : (
        <VideoProgressBar
          showControls={showControls}
          bufferedProgress={bufferedProgress}
          progress={progress}
          duration={duration}
          currentTime={currentTime}
          events={editor.events}
          onScrubStart={scrubStart}
          onScrubEnd={scrubEnd}
          onScrub={scrubTo}
          onHoverStart={handleHoverStart}
          onHoverEnd={handleHoverEnd}
        />
      )}

      {/* EDIT TOOLBAR + SELECTED-EVENT INSPECTOR */}
      {editor.isEditing && (
        <>
          <EventEditToolbar
            count={editor.draft.length}
            isSaving={editor.isSaving}
            canUndo={editor.canUndo}
            canRedo={editor.canRedo}
            canAdd={editor.canAddAtPlayhead}
            rotation={rotation}
            isRenamed={isNewName}
            pendingName={newName}
            onAdd={editor.addAtPlayhead}
            onUndo={editor.undo}
            onRedo={editor.redo}
            onRotate={handleRotation}
            onOpenRename={openRenameModal}
            onCancel={() => { requestDiscard(editor.cancel); }}
            onSave={requestCommit}
          />
          <EventInspector
            span={editor.selectedSpan}
            currentTime={currentTime}
            onNudge={editor.nudgeSelected}
            onSetToPlayhead={editor.setEdgeToPlayhead}
            onDelete={editor.deleteSelected}
          />
        </>
      )}

      {/* CONTROLS */}
      <VideoControls
        showControls={showControls}
        isPlaying={isPlaying}
        playbackRate={playbackRate}
        hasEvents={navEvents.length > 0}
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
          // While editing, the close action exits the editor first.
          if (editor.isEditing) {
            requestDiscard(editor.cancel);
            return;
          }
          onClose(disqualified, file.path, isNewName, newName, rotation);
        }}
        quality={quality}
        onChangeQuality={selectQuality}
        onEditEvents={() => {
          if (editor.isEditing) {
            requestDiscard(editor.cancel);
          } else {
            // Entering the editor pauses playback so the playhead is stable
            // while you line up event edges.
            if (isPlaying) togglePlay();
            editor.enter();
          }
        }}
        isEditingEvents={editor.isEditing}
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

      {/* DISCARD-EDITS CONFIRM (player-styled, on top of the editor) */}
      {discardAction && (
        <ConfirmDialog
          title="Discard event changes?"
          description="Your edits to the events have not been saved and will be lost."
          confirmLabel="Discard"
          destructive
          onConfirm={() => {
            const action = discardAction;
            setDiscardAction(null);
            action();
          }}
          onCancel={() => { setDiscardAction(null); }}
        />
      )}

      {/* DONE CONFIRM (edit mode): save + optional rename, then close the player.
          A rename makes it terminal (rotate + move into done/), so the copy and
          the commit differ from a plain in-place event save. */}
      {pendingCommit && (
        <ConfirmDialog
          title={
            pendingCommit.newName
              ? "Save events, rotate and move to Done?"
              : "Save event changes?"
          }
          description={
            pendingCommit.newName
              ? "The events will be modified and embedded, the video rotated if set, and the file moved into the done folder."
              : "The events will be modified and embedded in the video."
          }
          confirmLabel={pendingCommit.newName ? "Save & Done" : "Save"}
          onConfirm={() => { void runCommit(); }}
          onCancel={() => { setPendingCommit(null); }}
        />
      )}
    </div>
  );
};

export default VideoPlayerCompressModalV2;
