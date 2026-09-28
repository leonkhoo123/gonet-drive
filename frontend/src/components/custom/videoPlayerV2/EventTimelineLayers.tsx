import { Maximize2, ScanSearch, ZoomIn, ZoomOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatTime, type EventSpan } from "@/utils/videoPlayerV2Events";
import { CONTROL } from "./controlStyles";
import { clamp, type RulerTick, type ViewWindow } from "./timelineViewport";

interface TimelineRulerProps {
  duration: number;
  progressPct: number;
  canZoomOut: boolean;
  rulerTicks: RulerTick[];
  secondTicks: number[];
  timeToPct: (t: number) => number;
}

/** Top lane: ruler ticks plus the played-portion wash and playhead head. */
export function TimelineRuler({
  duration,
  progressPct,
  canZoomOut,
  rulerTicks,
  secondTicks,
  timeToPct,
}: TimelineRulerProps) {
  return (
    <div
      data-lane="ruler"
      className={`absolute top-0 inset-x-0 h-7 ${
        canZoomOut ? "cursor-grab active:cursor-grabbing" : ""
      }`}
    >
      <div
        className="absolute inset-y-0 left-0 bg-white/10 pointer-events-none"
        style={{ width: `${String(progressPct)}%` }}
      />
      <div className="absolute inset-x-0 bottom-0 h-px bg-white/15 pointer-events-none" />
      {secondTicks.map((t) => (
        <div
          key={`minor-${String(t)}`}
          className="absolute bottom-0 h-1.5 w-px bg-white/15 pointer-events-none"
          style={{ left: `${String(timeToPct(t))}%` }}
        />
      ))}
      {rulerTicks.map(({ t, align }) => (
        <div
          key={t}
          className="absolute bottom-0 pointer-events-none"
          style={{ left: `${String(timeToPct(t))}%` }}
        >
          <div className="absolute bottom-0 -translate-x-1/2 h-2.5 w-px bg-white/30" />
          <span
            className="absolute bottom-3 whitespace-nowrap text-[10px] leading-none tabular-nums text-white/60"
            style={{
              transform:
                align === "center"
                  ? "translateX(-50%)"
                  : align === "right"
                    ? "translateX(-100%)"
                    : "none",
            }}
          >
            {formatTime(t)}
          </span>
        </div>
      ))}
      {duration > 0 && (
        <div
          className="absolute top-0.5 -translate-x-1/2 h-1.5 w-1.5 rounded-full bg-white shadow-[0_0_6px_rgba(255,255,255,0.9)] pointer-events-none"
          style={{ left: `${String(progressPct)}%` }}
        />
      )}
    </div>
  );
}

interface EventRailProps {
  events: EventSpan[];
  duration: number;
  view: ViewWindow;
  selectedIndex: number | null;
  timeToPct: (t: number) => number;
}

/** Middle lane: the draggable event bands and their resize caps. */
export function EventRail({
  events,
  duration,
  view,
  selectedIndex,
  timeToPct,
}: EventRailProps) {
  return (
    <div
      data-lane="rail"
      className="absolute top-7 inset-x-0 h-12 border-t border-white/10 bg-white/[0.04]"
    >
      {/* Bands live inside the rail so their percentages track the lane, and
          hug its bottom: the ruler above stays band-free so a tap there seeks
          instead of selecting/dragging. */}
      {duration > 0 &&
        events.map(([start, end], index) => {
          if (end < view.start || start > view.end) return null;
          const startPct = timeToPct(start);
          const endPct = timeToPct(end);
          // Clamp the body to the window; the caps below only render when
          // their real edge is on screen, so a dragged handle stays truthful.
          const left = clamp(startPct, 0, 100);
          const right = clamp(endPct, 0, 100);
          const width = Math.max(right - left, 0.4);
          const showStartCap = startPct >= 0 && startPct <= 100;
          const showEndCap = endPct >= 0 && endPct <= 100;
          const isSelected = index === selectedIndex;
          return (
            <div
              key={`${String(start)}-${String(end)}-${String(index)}`}
              data-event-index={index}
              className={`absolute bottom-1 h-9 rounded-sm border cursor-grab active:cursor-grabbing transition-colors ${
                isSelected
                  ? // Above neighbours so an adjacent event's band cannot
                    // swallow this band's end cap at the shared boundary.
                    "z-10 border-emerald-200 bg-gradient-to-b from-emerald-300/60 to-emerald-500/40 ring-1 ring-emerald-200 shadow-[0_0_12px_rgba(52,211,153,0.55)]"
                  : "border-emerald-300/50 bg-gradient-to-b from-emerald-400/35 to-emerald-500/20 hover:from-emerald-400/45"
              }`}
              style={{ left: `${String(left)}%`, width: `${String(width)}%` }}
            >
              {/* Move affordance: a couple of grip dots in the middle. */}
              {!isSelected && width > 4 && (
                <div className="absolute inset-0 flex items-center justify-center gap-1 pointer-events-none">
                  <span className="h-3 w-px rounded bg-white/40" />
                  <span className="h-3 w-px rounded bg-white/40" />
                </div>
              )}

              {/* Resize caps, only on the selected band. The wrapper is a
                  wide, tall invisible hit target (24x48px); the white bar
                  inside is the slim visible grip. */}
              {isSelected && showStartCap && (
                <div
                  data-event-index={index}
                  data-handle="start"
                  className="absolute -left-3 top-1/2 -translate-y-1/2 h-12 w-6 cursor-ew-resize flex items-center justify-center"
                >
                  <span className="h-9 w-2 rounded-sm bg-white shadow-md" />
                </div>
              )}
              {isSelected && showEndCap && (
                <div
                  data-event-index={index}
                  data-handle="end"
                  className="absolute -right-3 top-1/2 -translate-y-1/2 h-12 w-6 cursor-ew-resize flex items-center justify-center"
                >
                  <span className="h-9 w-2 rounded-sm bg-white shadow-md" />
                </div>
              )}
            </div>
          );
        })}
    </div>
  );
}

interface TimelineOverviewProps {
  events: EventSpan[];
  duration: number;
  overviewStartPct: number;
  overviewEndPct: number;
}

/** Bottom lane: full clip with the current window highlighted. */
export function TimelineOverview({
  events,
  duration,
  overviewStartPct,
  overviewEndPct,
}: TimelineOverviewProps) {
  return (
    <div
      data-lane="overview"
      className="absolute bottom-0 inset-x-0 h-4 border-t border-white/10 bg-black/50"
    >
      {duration > 0 && (
        <>
          {events.map(([start, end], index) => {
            const left = clamp((start / duration) * 100, 0, 100);
            const right = clamp((end / duration) * 100, left, 100);
            return (
              <div
                key={`overview-${String(start)}-${String(end)}-${String(index)}`}
                className="absolute top-1 bottom-1 rounded-[1px] bg-emerald-400/50 pointer-events-none"
                style={{
                  left: `${String(left)}%`,
                  width: `${String(Math.max(right - left, 0.4))}%`,
                }}
              />
            );
          })}
          <div
            className="absolute inset-y-0 left-0 bg-black/60 pointer-events-none"
            style={{ width: `${String(overviewStartPct)}%` }}
          />
          <div
            className="absolute inset-y-0 right-0 bg-black/60 pointer-events-none"
            style={{ width: `${String(Math.max(0, 100 - overviewEndPct))}%` }}
          />
          <div
            className="absolute inset-y-0 border-x-2 border-white/80 bg-white/10 pointer-events-none"
            style={{
              left: `${String(overviewStartPct)}%`,
              width: `${String(Math.max(overviewEndPct - overviewStartPct, 0.5))}%`,
            }}
          />
        </>
      )}
    </div>
  );
}

interface TimelineZoomControlsProps {
  canZoomOut: boolean;
  canZoomIn: boolean;
  canZoomToSelection: boolean;
  onZoomOut: () => void;
  onZoomIn: () => void;
  onFit: () => void;
  onZoomToSelection: () => void;
}

/**
 * Zoom controls, floated above the ruler on the left so they never cover the
 * ruler's time labels on narrow screens.
 */
export function TimelineZoomControls({
  canZoomOut,
  canZoomIn,
  canZoomToSelection,
  onZoomOut,
  onZoomIn,
  onFit,
  onZoomToSelection,
}: TimelineZoomControlsProps) {
  return (
    <div
      className="absolute -top-7 left-1 z-40 flex items-center gap-0.5"
      onPointerDown={(e) => {
        e.stopPropagation();
      }}
    >
      <Button
        variant="ghost"
        size="icon"
        onClick={onZoomOut}
        disabled={!canZoomOut}
        title="Zoom out"
        aria-label="Zoom out"
        className={`${CONTROL} h-6 w-6 disabled:opacity-30`}
      >
        <ZoomOut className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={onZoomIn}
        disabled={!canZoomIn}
        title="Zoom in"
        aria-label="Zoom in"
        className={`${CONTROL} h-6 w-6 disabled:opacity-30`}
      >
        <ZoomIn className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={onFit}
        disabled={!canZoomOut}
        title="Fit whole clip"
        aria-label="Fit whole clip"
        className={`${CONTROL} h-6 w-6 disabled:opacity-30`}
      >
        <Maximize2 className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={onZoomToSelection}
        disabled={!canZoomToSelection}
        title="Zoom to selected event"
        aria-label="Zoom to selected event"
        className={`${CONTROL} h-6 w-6 disabled:opacity-30`}
      >
        <ScanSearch className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

interface TimelinePlayheadProps {
  duration: number;
  progressPct: number;
}

/** The vertical playhead line over the whole bar. */
export function TimelinePlayhead({
  duration,
  progressPct,
}: TimelinePlayheadProps) {
  if (!(duration > 0)) return null;
  return (
    <div
      className="absolute top-0 bottom-4 w-px z-20 bg-white/90 pointer-events-none"
      style={{ left: `${String(progressPct)}%` }}
    />
  );
}
