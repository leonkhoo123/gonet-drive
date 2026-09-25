import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Slider } from "@/components/ui/slider";
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
} from "lucide-react";
import type { VideoControlsProps } from "./types";

/** Selectable playback speeds, ordered fast -> slow (left -> right). */
const SPEEDS = [2, 1.5, 1.25, 1, 0.75, 0.5, 0.25] as const;

/** Shared look for the control column, derived from the speed panel. */
const GLASS =
  "border border-white/20 bg-black/60 backdrop-blur-sm text-white hover:bg-black/70";
/** Same glass treatment, tinted for the rename action. */
const GLASS_GREEN =
  "border border-white/20 bg-green-600/50 backdrop-blur-sm text-white hover:bg-green-600/70";
/** Same glass treatment, tinted for the disqualified action. */
const GLASS_RED =
  "border border-white/20 bg-red-600/50 backdrop-blur-sm text-white hover:bg-red-600/70";
/** Same glass treatment, tinted for the rotate action. */
const GLASS_YELLOW =
  "border border-white/20 bg-yellow-500/50 backdrop-blur-sm text-white hover:bg-yellow-500/70";

/** Which collapsible flyout is currently open. */
type Panel = "actions" | "speed" | null;

/** Right-hand vertical control column. */
export function VideoControls({
  showControls,
  isPlaying,
  playbackRate,
  hasEvents,
  onPressStart,
  onPressEnd,
  onSkip,
  onPrevEvent,
  onNextEvent,
  onTogglePlay,
  onChangeSpeed,
  onOpenRename,
  onToggleDisqualified,
  onRotate,
  onClose,
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

  const speedIndex = SPEEDS.reduce(
    (best, speed, index) =>
      Math.abs(speed - playbackRate) < Math.abs(SPEEDS[best] - playbackRate)
        ? index
        : best,
    0
  );

  const handleSpeedChange = (values: number[]) => {
    onChangeSpeed(SPEEDS[values[0]]);
  };

  return (
    <div
      className={`absolute right-0 top-0 bottom-12 flex flex-col p-1 lg:p-2 transition-opacity duration-300 ${
        showControls ? "opacity-100" : "opacity-0 pointer-events-none"
      }`}
      onMouseDown={onPressStart}
      onMouseUp={onPressEnd}
      onTouchStart={onPressStart}
      onTouchEnd={onPressEnd}
    >
      <div className="m-auto flex flex-col gap-1 lg:gap-2 h-full max-h-[100%] overflow-y-auto scrollbar-hide w-20 lg:w-24 py-2 justify-center">
        {/* previous detected event - only rendered when metadata exists */}
        {hasEvents && (
          <Button
            variant="ghost"
            onClick={onPrevEvent}
            title="Previous event (P or Shift+,)"
            className={`${GLASS} w-full flex-1 min-h-[32px] max-h-12 px-1`}
          >
            Evt <ChevronFirst className="h-4 w-4 ml-1" />
          </Button>
        )}

        <Button
          variant="ghost"
          onClick={() => {
            onSkip(-1);
          }}
          className={`${GLASS} w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          <SkipBack className="h-4 w-4 mr-1" /> 1s
        </Button>

        {/* forward 3 sec */}
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(3);
          }}
          className={`${GLASS} w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* forward 1 sec */}
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(1);
          }}
          className={`${GLASS} w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* next detected event - only rendered when metadata exists */}
        {hasEvents && (
          <Button
            variant="ghost"
            onClick={onNextEvent}
            title="Next event (N or Shift+.)"
            className={`${GLASS} w-full flex-1 min-h-[32px] max-h-12 px-1`}
          >
            Evt <ChevronLast className="h-4 w-4 ml-1" />
          </Button>
        )}

        {/* play/pause */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onTogglePlay}
          className={`${GLASS} w-full flex-[2] min-h-[40px] max-h-24`}
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
          className={`${GLASS} relative w-full flex-1 min-h-[32px] max-h-12 px-1`}
        >
          {/* Chevron pinned left; rotates when the speed slider is open */}
          <ChevronLeft
            className={`absolute left-2 h-4 w-4 sm:h-5 sm:w-5 text-gray-500 transition-transform duration-300 ${
              openPanel === "speed" ? "rotate-180" : ""
            }`}
          />
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate}x
        </Button>

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
          className={`${GLASS} relative w-full flex-1 min-h-[32px] max-h-12`}
        >
          {/* Chevron pinned to the left; rotates between < and > when toggled */}
          <ChevronLeft
            className={`absolute left-2 h-4 w-4 sm:h-5 sm:w-5 text-gray-500 transition-transform duration-300 ${
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
          className={`${GLASS} w-full flex-1 min-h-[32px] max-h-12`}
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
        <div className="flex w-[calc(100vw-7rem)] max-w-sm flex-col gap-2 rounded-md border border-white/20 bg-black/60 px-3 py-2 text-white shadow-lg backdrop-blur-sm">
          <div className="flex items-center justify-between text-xs">
            <span className="opacity-80">Speed</span>
            <span className="font-semibold tabular-nums">{playbackRate}x</span>
          </div>
          <Slider
            min={0}
            max={SPEEDS.length - 1}
            step={1}
            value={[speedIndex]}
            onValueChange={handleSpeedChange}
            aria-label="Playback speed"
            className="[&_[data-slot=slider-track]]:h-2 [&_[data-slot=slider-track]]:bg-white/25 [&_[data-slot=slider-range]]:bg-white/90 [&_[data-slot=slider-thumb]]:size-5 [&_[data-slot=slider-thumb]]:border-white [&_[data-slot=slider-thumb]]:bg-white [&_[data-slot=slider-thumb]]:shadow-md"
          />
          <div className="relative h-3 text-[10px] tabular-nums text-white/70">
            {SPEEDS.map((speed, index) => (
              // Radix insets the 20px thumb by 10px on each side, so its
              // center travels from +10px to (100% - 10px). Match that here.
              <span
                key={speed}
                className="absolute top-0 -translate-x-1/2"
                style={{
                  left: `calc(10px + (100% - 20px) * ${String(
                    index / (SPEEDS.length - 1)
                  )})`,
                }}
              >
                {speed}
              </span>
            ))}
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
        {/* --- disqualified Button --- */}
        <Button
          variant="ghost"
          onClick={() => {
            runAction(onToggleDisqualified);
          }}
          title="Disqualify"
          aria-label="Disqualify"
          className={`${GLASS_RED} h-12 w-14 p-0`}
        >
          <ListX className="size-5" />
        </Button>

        {/* --- Rotation Button --- */}
        <Button
          variant="ghost"
          onClick={() => {
            runAction(onRotate);
          }}
          title="Rotate"
          aria-label="Rotate"
          className={`${GLASS_YELLOW} h-12 w-14 p-0`}
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
          className={`${GLASS_GREEN} h-12 w-14 p-0`}
        >
          <TextCursorInput className="size-5" />
        </Button>
      </div>
    </div>
  );
}
