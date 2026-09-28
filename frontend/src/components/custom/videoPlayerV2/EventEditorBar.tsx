import { useMemo, useRef, type PointerEvent as ReactPointerEvent } from "react";
import { MIN_EVENT_SECONDS } from "@/hooks/useVideoPlayerV2/useVideoEventEditor";
import { formatTime, type EventSpan } from "@/utils/videoPlayerV2Events";
import type { EventEditorBarProps } from "./types";

type DragMode = "move" | "start" | "end";

interface DragState {
  index: number;
  mode: DragMode;
  startX: number;
  origin: EventSpan;
  /** Boundary (seconds) the dragged edge is currently stuck to, if any. */
  stuckTo: number | null;
}

/** Pixels past a stuck boundary before the drag breaks free into an overlap. */
const BREAK_PX = 24;
/** Magnetic snap distance (px) for moving a whole band toward a neighbor. */
const MOVE_SNAP_PX = 10;

/** Round intervals (seconds) the ruler labels prefer, so times read formally. */
const RULER_STEPS = [1, 2, 5, 10, 15, 20, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 10800, 21600];
/** Aim for roughly this many labels; the step snaps up to the next nice value. */
const RULER_TARGET_LABELS = 6;

type TickAlign = "left" | "center" | "right";

interface RulerTick {
  t: number;
  align: TickAlign;
}

/** Pick a nice interval for the clip length and lay out labelled ticks. */
function buildRulerTicks(duration: number): RulerTick[] {
  if (!(duration > 0)) return [];
  const raw = duration / RULER_TARGET_LABELS;
  const step = RULER_STEPS.find((s) => s >= raw) ?? Math.ceil(raw);

  const ticks: RulerTick[] = [];
  for (let t = 0; t <= duration + 1e-6; t += step) {
    ticks.push({ t: Math.min(t, duration), align: "center" });
  }
  // Label the clip end too, unless it would crowd the previous tick.
  if (duration - ticks[ticks.length - 1].t > step * 0.4) {
    ticks.push({ t: duration, align: "center" });
  }
  ticks[0].align = "left";
  ticks[ticks.length - 1].align = "right";
  return ticks;
}

const clamp = (value: number, lo: number, hi: number): number =>
  Math.min(hi, Math.max(lo, value));

/**
 * Edit-mode timeline, laid out as two lanes:
 *
 *   ┌ seek ruler ─────────────────────────────┐  tap/drag to move the playhead
 *   ├ event rail ─────────────────────────────┤  green bands, drag to edit
 *   └──────────────────────────────────────────┘
 *
 * The ruler stays band-free, so tapping anywhere on it seeks even when an event
 * spans that moment. Resize caps stick to the neighbouring event's edge and only
 * break through into an overlap after a further 24px push (the host's release
 * trim cuts the overlap); moving a band snaps magnetically to nearby edges. All
 * input goes through Pointer Events (mouse/touch/pen share one path) and
 * `touch-action: none` stops the page from scrolling.
 */
export function EventEditorBar({
  events,
  duration,
  currentTime,
  selectedIndex,
  onSelect,
  onSeek,
  onBeginChange,
  onPreviewChange,
  onEndChange,
}: EventEditorBarProps) {
  const barRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<DragState | null>(null);

  const pct = (t: number): number =>
    duration > 0 ? clamp((t / duration) * 100, 0, 100) : 0;

  const rulerTicks = useMemo(() => buildRulerTicks(duration), [duration]);

  const handlePointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    const target = (e.target as HTMLElement).closest<HTMLElement>(
      "[data-event-index]"
    );

    if (!target) {
      // Ruler / empty rail: seek to the tapped position, snapped to the 1s grid
      // like every other edit, and drop the selection.
      const rect = barRef.current?.getBoundingClientRect();
      if (rect && duration > 0) {
        const seconds = ((e.clientX - rect.left) / rect.width) * duration;
        onSeek(clamp(Math.round(seconds), 0, duration));
      }
      onSelect(null);
      return;
    }

    const index = Number(target.dataset.eventIndex);
    const mode = (target.dataset.handle as DragMode | undefined) ?? "move";
    const origin = events[index];

    e.preventDefault();
    e.stopPropagation();
    barRef.current?.setPointerCapture(e.pointerId);
    dragRef.current = {
      index,
      mode,
      startX: e.clientX,
      origin: [origin[0], origin[1]],
      stuckTo: null,
    };
    onSelect(index);
    onBeginChange();
  };

  const handlePointerMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    const rect = barRef.current?.getBoundingClientRect();
    if (!drag || !rect || duration <= 0) return;

    const pxToSec = duration / rect.width;
    const breakSec = BREAK_PX * pxToSec;
    const snapSec = MOVE_SNAP_PX * pxToSec;
    // 1s resolution: snap every edge to a whole second (jerky is fine).
    const q = (seconds: number): number => Math.round(seconds);

    const dt = ((e.clientX - drag.startX) / rect.width) * duration;
    const [originStart, originEnd] = drag.origin;
    const originStartQ = q(originStart);
    const originEndQ = q(originEnd);
    // Neighbors are fixed for the duration of this gesture (the draft is not
    // re-sorted until release), so the boundary a cap sticks to is stable.
    const prevEnd = drag.index > 0 ? q(events[drag.index - 1][1]) : null;
    const nextStart =
      drag.index < events.length - 1 ? q(events[drag.index + 1][0]) : null;

    let start = originStartQ;
    let end = originEndQ;

    if (drag.mode === "move") {
      const length = Math.max(MIN_EVENT_SECONDS, q(originEnd - originStart));
      start = clamp(q(originStart + dt), 0, Math.max(0, duration - length));
      end = start + length;

      // Magnetic snap: ease to whichever neighbor edge is closest.
      const dPrev = prevEnd !== null ? Math.abs(start - prevEnd) : Infinity;
      const dNext = nextStart !== null ? Math.abs(end - nextStart) : Infinity;
      if (dPrev <= snapSec && dPrev <= dNext && prevEnd !== null) {
        start = prevEnd;
        end = start + length;
      } else if (dNext <= snapSec && nextStart !== null) {
        end = nextStart;
        start = end - length;
      }
      start = clamp(start, 0, Math.max(0, duration - length));
      end = start + length;
    } else if (drag.mode === "start") {
      const raw = clamp(q(originStart + dt), 0, originEndQ - MIN_EVENT_SECONDS);

      // Stick to the previous event's end; keep pushing to break free into an
      // overlap, which the release-time trim resolves by cutting that end.
      start = raw;
      if (prevEnd !== null && originEndQ - prevEnd >= MIN_EVENT_SECONDS) {
        if (drag.stuckTo !== null) {
          if (raw < drag.stuckTo - breakSec || raw > drag.stuckTo) {
            drag.stuckTo = null;
            start = raw;
          } else {
            start = drag.stuckTo;
          }
        } else if (raw <= prevEnd) {
          drag.stuckTo = prevEnd;
          start = prevEnd;
        }
      }
      start = clamp(start, 0, originEndQ - MIN_EVENT_SECONDS);
      end = originEndQ;
    } else {
      const raw = clamp(q(originEnd + dt), originStartQ + MIN_EVENT_SECONDS, duration);

      // Stick to the next event's start; keep pushing to break free into an
      // overlap, which the release-time trim resolves by cutting this end.
      end = raw;
      if (nextStart !== null && nextStart - originStartQ >= MIN_EVENT_SECONDS) {
        if (drag.stuckTo !== null) {
          if (raw > drag.stuckTo + breakSec || raw < drag.stuckTo) {
            drag.stuckTo = null;
            end = raw;
          } else {
            end = drag.stuckTo;
          }
        } else if (raw >= nextStart) {
          drag.stuckTo = nextStart;
          end = nextStart;
        }
      }
      end = clamp(end, originStartQ + MIN_EVENT_SECONDS, duration);
      start = originStartQ;
    }

    onPreviewChange(drag.index, [start, end]);
  };

  const handlePointerUp = () => {
    if (!dragRef.current) return;
    dragRef.current = null;
    onEndChange();
  };

  return (
    // No overflow clipping: the resize caps sit slightly outside their band so
    // an event at 0%/100% keeps a grabbable handle.
    <div
      ref={barRef}
      className="absolute bottom-12 left-0 w-full h-20 z-30 touch-none select-none border-t border-white/15 bg-black/60 shadow-[0_-6px_18px_rgba(0,0,0,0.55)]"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
    >
      {/* RULER (top, always seekable) */}
      <div className="absolute top-0 inset-x-0 h-8 pointer-events-none">
        <div
          className="absolute inset-y-0 left-0 bg-white/10"
          style={{ width: `${String(pct(currentTime))}%` }}
        />
        <div className="absolute inset-x-0 bottom-0 h-px bg-white/15" />
        {rulerTicks.map(({ t, align }) => (
          <div key={t} className="absolute bottom-0" style={{ left: `${String(pct(t))}%` }}>
            <div className="absolute bottom-0 -translate-x-1/2 h-3 w-px bg-white/30" />
            <span
              className="absolute bottom-3.5 whitespace-nowrap text-[10px] leading-none tabular-nums text-white/60"
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
            className="absolute top-0.5 -translate-x-1/2 h-1.5 w-1.5 rounded-full bg-white shadow-[0_0_6px_rgba(255,255,255,0.9)]"
            style={{ left: `${String(pct(currentTime))}%` }}
          />
        )}
      </div>

      {/* EVENT RAIL (bottom) */}
      <div className="absolute bottom-0 inset-x-0 h-12 border-t border-white/10 bg-white/[0.04]" />

      {/* Bands */}
      {duration > 0 &&
        events.map(([start, end], index) => {
          const left = pct(start);
          const width = Math.max(pct(end) - left, 0.4);
          const isSelected = index === selectedIndex;
          return (
            <div
              key={`${String(start)}-${String(end)}-${String(index)}`}
              data-event-index={index}
              // Bands hug the bottom of the track: the ruler above stays
              // band-free so a tap there seeks instead of selecting/dragging.
              className={`absolute bottom-1.5 h-9 rounded-sm border cursor-grab active:cursor-grabbing transition-colors ${
                isSelected
                  ? // Above neighbours so an adjacent event's band cannot swallow
                    // this band's end cap at the shared boundary.
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

              {/* Resize caps, only on the selected band. The wrapper is a wide,
                  tall invisible hit target (24x48px); the white bar inside is
                  the slim visible grip. */}
              {isSelected && (
                <>
                  <div
                    data-event-index={index}
                    data-handle="start"
                    className="absolute -left-3 top-1/2 -translate-y-1/2 h-12 w-6 cursor-ew-resize flex items-center justify-center"
                  >
                    <span className="h-9 w-2 rounded-sm bg-white shadow-md" />
                  </div>
                  <div
                    data-event-index={index}
                    data-handle="end"
                    className="absolute -right-3 top-1/2 -translate-y-1/2 h-12 w-6 cursor-ew-resize flex items-center justify-center"
                  >
                    <span className="h-9 w-2 rounded-sm bg-white shadow-md" />
                  </div>
                </>
              )}
            </div>
          );
        })}

      {/* Playhead */}
      {duration > 0 && (
        <div
          className="absolute top-0 bottom-0 w-px z-20 bg-white/90 pointer-events-none"
          style={{ left: `${String(pct(currentTime))}%` }}
        />
      )}
    </div>
  );
}
