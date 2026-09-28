import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Play,
  Pause,
  Zap,
  SkipBack,
  SkipForward,
  ChevronFirst,
  ChevronLast,
  LogOut,
  TextCursorInput,
  ListX,
  RotateCw,
  ChevronLeft,
  Pencil,
  ListVideo,
  Shuffle,
  CircleOff,
  Gauge,
} from "lucide-react";
import type { AutoPlayMode } from "@/hooks/useVideoPlayerV2/useVideoAutoPlay";
import type { VideoQuality } from "@/hooks/useVideoPlayerV2/useVideoPlaybackController";
import { PLAYBACK_SPEEDS } from "@/utils/videoSpeed";
import type { VideoControlsProps } from "./types";
import {
  CONTROL,
  CONTROL_ACTIVE,
  CONTROL_BLUE,
  CONTROL_GREEN,
  CONTROL_RED,
  CONTROL_YELLOW,
  PANEL,
} from "./controlStyles";

/** Selectable playback speeds, ordered fast -> slow (left -> right). */
const SPEEDS = [...PLAYBACK_SPEEDS].reverse();

/** Per-mode look and copy for the cycling auto-play button. */
const AUTO_PLAY_LOOK: Record<
  AutoPlayMode,
  { label: string; icon: typeof ListVideo; className: string }
> = {
  off: { label: "Off", icon: CircleOff, className: CONTROL },
  auto: { label: "Auto", icon: ListVideo, className: CONTROL_GREEN },
  shuffle: { label: "Shuffle", icon: Shuffle, className: CONTROL_BLUE },
};

/** Selectable qualities, in the order shown in the flyout. */
const QUALITY_OPTIONS: { value: VideoQuality; label: string }[] = [
  { value: "original", label: "Original" },
  { value: 1080, label: "1080p" },
  { value: 720, label: "720p" },
  { value: 480, label: "480p" },
];

/** Short label for the quality button. */
const qualityLabel = (quality: VideoQuality): string =>
  quality === "original" ? "Orig" : `${String(quality)}p`;

/** Which collapsible flyout is currently open. */
type Panel = "actions" | "speed" | "quality" | null;

/** Right-hand vertical control column. */
export function VideoControls({
  showControls,
  isPlaying,
  playbackRate,
  hasEvents,
  autoPlayMode,
  hasNext,
  shuffleRemaining,
  onPressStart,
  onPressEnd,
  onHoverStart,
  onHoverEnd,
  onSkip,
  onPrevEvent,
  onNextEvent,
  onTogglePlay,
  onCycleAutoPlayMode,
  onChangeSpeed,
  onOpenRename,
  onToggleDisqualified,
  onRotate,
  onClose,
  quality,
  onChangeQuality,
  onEditEvents,
  isEditingEvents,
}: VideoControlsProps) {
  /** Rename / disqualified / rotate plus the speed slider are collapsible. */
  const [openPanel, setOpenPanel] = useState<Panel>(null);
  /** Viewport coordinates for the fixed flyout, measured from its button. */
  const [flyoutPos, setFlyoutPos] = useState<{
    top: number;
    right: number;
  } | null>(null);
  const actionsRef = useRef<HTMLButtonElement>(null);
  const speedRef = useRef<HTMLButtonElement>(null);
  const qualityRef = useRef<HTMLButtonElement>(null);

  // Always start collapsed each time the overlay hides and reappears.
  useEffect(() => {
    if (!showControls) setOpenPanel(null);
  }, [showControls]);

  /**
   * Toggle a flyout panel. The flyout is rendered with `position: fixed`
   * (see below) so the column's vertical scroll/overflow cannot clip it; we
   * measure the trigger so it lines up with it and opens to the left.
   */
  const togglePanel = (panel: Exclude<Panel, null>, ref: typeof actionsRef) => {
    if (openPanel === panel) {
      setOpenPanel(null);
      return;
    }
    const rect = ref.current?.getBoundingClientRect();
    if (rect) {
      setFlyoutPos({
        top: rect.top + rect.height / 2,
        // 8px gap between the trigger's left edge and the flyout's right edge.
        right: window.innerWidth - rect.left + 8,
      });
    }
    setOpenPanel(panel);
  };

  /** Collapse the flyout after an action is chosen. */
  const runAction = (action: () => void) => {
    action();
    setOpenPanel(null);
  };

  const autoPlayLook = AUTO_PLAY_LOOK[autoPlayMode];
  const autoPlayTitle =
    autoPlayMode === "off"
      ? "Auto-play next video: off"
      : autoPlayMode === "auto"
        ? hasNext
          ? "Auto-play next video: on"
          : "Auto-play: on (last video)"
        : shuffleRemaining > 0
          ? `Shuffle: ${String(shuffleRemaining)} left`
          : "Shuffle: complete";

  return (
    <div
      className={`absolute right-0 top-0 bottom-12 flex flex-col p-1 lg:p-2 transition-opacity duration-300 ${
        showControls ? "opacity-100" : "opacity-0 pointer-events-none"
      }`}
      onMouseDown={onPressStart}
      onMouseUp={onPressEnd}
      onMouseEnter={onHoverStart}
      onMouseLeave={onHoverEnd}
      onTouchStart={onPressStart}
      onTouchEnd={onPressEnd}
    >
      {/* Auto-play / shuffle toggle pinned to the top of the column */}
      <div className="flex shrink-0 justify-center">
        <Button
          variant="ghost"
          onClick={onCycleAutoPlayMode}
          aria-label={`Play mode: ${autoPlayLook.label}. Click to change.`}
          title={autoPlayTitle}
          className={`${autoPlayLook.className} w-20 lg:w-24 h-10 px-1 text-xs`}
        >
          <autoPlayLook.icon className="h-4 w-4 mr-1" />
          {autoPlayLook.label}
        </Button>
      </div>

      <div className="flex-1 min-h-0 flex flex-col gap-1 lg:gap-2 overflow-y-auto scrollbar-hide w-20 lg:w-24 py-2 justify-center">
        {/* previous detected event - only rendered when metadata exists */}
        {hasEvents && (
          <Button
            variant="ghost"
            onClick={onPrevEvent}
            title="Previous event (Ctrl+←)"
            className={`${CONTROL} w-full flex-1 min-h-[32px] max-h-12 px-1`}
          >
            Evt <ChevronFirst className="h-4 w-4 ml-1" />
          </Button>
        )}

        <Button
          variant="ghost"
          onClick={() => {
            onSkip(-1);
          }}
          className={`${CONTROL} w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          <SkipBack className="h-4 w-4 mr-1" /> 1s
        </Button>

        {/* forward 3 sec */}
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(3);
          }}
          className={`${CONTROL} w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* forward 1 sec */}
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(1);
          }}
          className={`${CONTROL} w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* next detected event - only rendered when metadata exists */}
        {hasEvents && (
          <Button
            variant="ghost"
            onClick={onNextEvent}
            title="Next event (Ctrl+→)"
            className={`${CONTROL} w-full flex-1 min-h-[32px] max-h-12 px-1`}
          >
            Evt <ChevronLast className="h-4 w-4 ml-1" />
          </Button>
        )}

        {/* play/pause */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onTogglePlay}
          className={`${CONTROL} w-full flex-[2] min-h-[40px] max-h-24`}
        >
          {isPlaying ? (
            <Pause className="h-6 w-6" />
          ) : (
            <Play className="h-6 w-6" />
          )}
        </Button>

        {/* speed - opens the collapsible speed slider */}
        <Button
          ref={speedRef}
          variant="ghost"
          onClick={() => {
            togglePanel("speed", speedRef);
          }}
          aria-expanded={openPanel === "speed"}
          aria-label="Playback speed"
          title="Playback speed"
          className={`${CONTROL} relative w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate}x
        </Button>

        {/* quality - opens the resolution flyout (only for the compress player) */}
        {onChangeQuality && quality !== undefined && (
          <Button
            ref={qualityRef}
            variant="ghost"
            onClick={() => {
              togglePanel("quality", qualityRef);
            }}
            aria-expanded={openPanel === "quality"}
            aria-label="Playback quality"
            title="Playback quality"
            className={`${CONTROL} relative w-full flex-1 min-h-[32px] max-h-12 px-1`}
          >
            <Gauge className="h-4 w-4 mr-1" />
            {qualityLabel(quality)}
          </Button>
        )}

        {/* --- Collapsible actions toggle (Rename / Disqualify / Rotate) --- */}
        <Button
          ref={actionsRef}
          variant="ghost"
          size="icon"
          onClick={() => {
            togglePanel("actions", actionsRef);
          }}
          aria-expanded={openPanel === "actions"}
          aria-label={openPanel === "actions" ? "Hide actions" : "Show actions"}
          title={openPanel === "actions" ? "Hide actions" : "Show actions"}
          className={`${CONTROL} relative w-full flex-1 min-h-[32px] max-h-12`}
        >
          {/* Chevron pinned to the left; rotates between < and > when toggled */}
          <ChevronLeft
            className={`absolute left-2 h-4 w-4 sm:h-5 sm:w-5 text-white/60 transition-transform duration-300 ${
              openPanel === "actions" ? "rotate-180" : ""
            }`}
          />
          {/* Pencil hints that this toggle reveals edit actions */}
          <Pencil className="size-4 opacity-80 sm:size-5" />
        </Button>

        {/* --- close --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onClose}
          className={`${CONTROL} w-full flex-1 min-h-[32px] max-h-12`}
        >
          <LogOut className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>
      </div>

      {/* -------- Speed flyout (fixed so the column cannot clip it) -------- */}
      <div
        style={flyoutPos ? { top: flyoutPos.top, right: flyoutPos.right } : undefined}
        className={`fixed z-40 -translate-y-1/2 transition-all duration-300 ${
          openPanel === "speed"
            ? "opacity-100 translate-x-0 pointer-events-auto"
            : "opacity-0 translate-x-3 pointer-events-none"
        }`}
      >
        <div className={`flex w-[calc(100vw-7rem)] max-w-sm flex-col gap-2 rounded-md px-3 py-2 ${PANEL}`}>
          <div className="flex items-center justify-between text-xs">
            <span className="opacity-80">Speed</span>
            <span className="font-semibold tabular-nums">{playbackRate}x</span>
          </div>
          {/* One button per speed step — easier to hit than the slider on touch. */}
          <div className="grid grid-cols-4 gap-1 sm:grid-cols-7">
            {SPEEDS.map((speed) => {
              const isActive = speed === playbackRate;
              return (
                <Button
                  key={speed}
                  variant="ghost"
                  onClick={() => {
                    runAction(() => {
                      onChangeSpeed(speed);
                    });
                  }}
                  aria-pressed={isActive}
                  aria-label={`Set playback speed to ${String(speed)}x`}
                  className={`h-9 px-0 text-xs tabular-nums ${
                    isActive ? CONTROL_ACTIVE : CONTROL
                  }`}
                >
                  {speed}x
                </Button>
              );
            })}
          </div>
        </div>
      </div>

      {/* -------- Quality flyout (fixed so the column cannot clip it) -------- */}
      <div
        style={flyoutPos ? { top: flyoutPos.top, right: flyoutPos.right } : undefined}
        className={`fixed z-40 -translate-y-1/2 transition-all duration-300 ${
          openPanel === "quality"
            ? "opacity-100 translate-x-0 pointer-events-auto"
            : "opacity-0 translate-x-3 pointer-events-none"
        }`}
      >
        <div className={`flex w-[calc(100vw-7rem)] max-w-sm flex-col gap-2 rounded-md px-3 py-2 ${PANEL}`}>
          <div className="flex items-center justify-between text-xs">
            <span className="opacity-80">Quality</span>
            <span className="font-semibold">
              {quality !== undefined ? qualityLabel(quality) : ""}
            </span>
          </div>
          <div className="grid grid-cols-4 gap-1">
            {QUALITY_OPTIONS.map((option) => {
              const isActive = option.value === quality;
              return (
                <Button
                  key={String(option.value)}
                  variant="ghost"
                  onClick={() => {
                    onChangeQuality?.(option.value);
                    setOpenPanel(null);
                  }}
                  aria-pressed={isActive}
                  aria-label={`Set quality to ${option.label}`}
                  className={`h-9 px-0 text-xs ${isActive ? CONTROL_ACTIVE : CONTROL}`}
                >
                  {option.label}
                </Button>
              );
            })}
          </div>
        </div>
      </div>

      {/* -------- Actions flyout (fixed so the column cannot clip it) -------- */}
      <div
        style={flyoutPos ? { top: flyoutPos.top, right: flyoutPos.right } : undefined}
        className={`fixed z-40 -translate-y-1/2 flex flex-row gap-1 lg:gap-2 transition-all duration-300 ${
          openPanel === "actions"
            ? "opacity-100 translate-x-0 pointer-events-auto"
            : "opacity-0 translate-x-3 pointer-events-none"
        }`}
      >
        {/* --- edit events (entry point to the event editor) --- */}
        {onEditEvents && (
          <Button
            variant="ghost"
            onClick={() => {
              runAction(onEditEvents);
            }}
            title="Edit events"
            aria-label="Edit events"
            aria-pressed={isEditingEvents}
            className={`${CONTROL_BLUE} h-12 w-14 p-0 ${
              isEditingEvents ? "ring-2 ring-white" : ""
            }`}
          >
            <Pencil className="size-5" />
          </Button>
        )}

        {/* --- disqualified Button --- */}
        <Button
          variant="ghost"
          onClick={() => {
            runAction(onToggleDisqualified);
          }}
          title="Disqualify"
          aria-label="Disqualify"
          className={`${CONTROL_RED} h-12 w-14 p-0`}
        >
          <ListX className="size-5" />
        </Button>

        {/* --- Rotation Button --- */}
        <Button
          variant="ghost"
          onClick={() => {
            // Rotation is repeatable: keep the flyout open so it can be tapped again.
            onRotate();
          }}
          title="Rotate"
          aria-label="Rotate"
          className={`${CONTROL_YELLOW} h-12 w-14 p-0`}
        >
          <RotateCw className="size-5" />
        </Button>

        {/* --- Rename Button --- */}
        <Button
          variant="ghost"
          onClick={() => {
            runAction(onOpenRename);
          }}
          title="Rename"
          aria-label="Rename"
          className={`${CONTROL_GREEN} h-12 w-14 p-0`}
        >
          <TextCursorInput className="size-5" />
        </Button>
      </div>
    </div>
  );
}
