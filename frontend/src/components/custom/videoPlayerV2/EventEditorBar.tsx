import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { Maximize2, ScanSearch, ZoomIn, ZoomOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MIN_EVENT_SECONDS } from "@/hooks/useVideoPlayerV2/useVideoEventEditor";
import { formatTime, type EventSpan } from "@/utils/videoPlayerV2Events";
import { CONTROL } from "./controlStyles";
import {
  buildRulerTicks,
  buildSecondTicks,
  centerWindowOn,
  clamp,
  minWindowSpan,
  normalizeWindow,
  panWindow,
  zoomWindowAt,
  type ViewWindow,
} from "./timelineViewport";
import type { EventEditorBarProps } from "./types";

type DragMode = "move" | "start" | "end";

interface DragState {
  /** Pointer that owns this drag, so a second finger can cancel it cleanly. */
  pointerId: number;
  index: number;
  mode: DragMode;
  startX: number;
  origin: EventSpan;
  /** Boundary (seconds) the dragged edge is currently stuck to, if any. */
  stuckTo: number | null;
}

/** Two-finger zoom/pan gesture, anchored on the initial finger midpoint. */
interface PinchState {
  startDist: number;
  startSpan: number;
  /** Clip time under the initial midpoint; pinned there as the fingers move. */
  anchorTime: number;
}

/** Single-pointer pan: the ruler on desktop, or the overview window. */
interface PanState {
  pointerId: number;
  kind: "ruler" | "overview";
  startX: number;
  startStart: number;
  /** Seconds across the full bar width (window span, or clip length). */
  secondsPerWidth: number;
  /** Window span held constant while panning. */
  viewSpan: number;
  moved: boolean;
  tapTime: number;
}

/** Pixels past a stuck boundary before the drag breaks free into an overlap. */
const BREAK_PX = 24;
/** Magnetic snap distance (px) for moving a whole band toward a neighbor. */
const MOVE_SNAP_PX = 10;
/** Movement past this (px) turns a ruler press into a pan instead of a tap. */
const PAN_TAP_PX = 6;
/** Zoom applied per +/− button press. */
const BUTTON_ZOOM = 1.6;
/** Draw the whole-second grid only when a second is at least this many px wide. */
const MINOR_TICK_MIN_PX_PER_SEC = 8;

const sameWindow = (a: ViewWindow, b: ViewWindow): boolean =>
  Math.abs(a.start - b.start) < 1e-6 && Math.abs(a.end - b.end) < 1e-6;

/**
 * Edit-mode timeline, laid out as three lanes:
 *
 *   ┌ ruler ──────────────────────────────────┐  tap/drag to seek or pan
 *   ├ event rail ─────────────────────────────┤  green bands, drag to edit
 *   └ overview ───────────────────────────────┘  full clip + current window
 *
 * Unlike the old version this shows a movable window `[start, end]` rather than
 * the whole clip, so a short event stays editable on a narrow phone: pinch to
 * zoom (up to 64×), two-finger drag or ruler drag to pan, ⌘/Ctrl+wheel (or the
 * +/− buttons) on desktop. Every edit snaps to the 1s grid, resize caps stick
 * to the neighbouring edge and only break through after 24px, and a band snaps
 * magnetically to nearby edges. All input uses Pointer Events (mouse/touch/pen
 * share one path) and `touch-action: none` stops the page from scrolling.
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
  const panRef = useRef<PanState | null>(null);
  const pinchRef = useRef<PinchState | null>(null);
  const pointersRef = useRef<Map<number, { x: number; y: number }>>(new Map());

  // Visible window. A fresh edit session starts fitted; `normalizeWindow`
  // re-fits it if the source duration changes mid-session.
  const [view, setView] = useState<ViewWindow>(() =>
    duration > 0 ? { start: 0, end: duration } : { start: 0, end: 0 }
  );
  const [barWidth, setBarWidth] = useState(0);

  const windowSpan = view.end - view.start;
  const minSpan = minWindowSpan(duration);

  const timeToPct = useCallback(
    (t: number): number =>
      windowSpan > 0 ? ((t - view.start) / windowSpan) * 100 : 0,
    [view.start, windowSpan]
  );

  /** Pointer/client-x → absolute clip seconds for the current window. */
  const clientXToTime = useCallback(
    (clientX: number, rect: DOMRect): number => {
      if (!(windowSpan > 0) || rect.width <= 0) return view.start;
      return view.start + ((clientX - rect.left) / rect.width) * windowSpan;
    },
    [view.start, windowSpan]
  );

  useEffect(() => {
    setView((v) => normalizeWindow(v, duration));
  }, [duration]);

  // Track the bar width so the second-level grid only appears when it is legible.
  useEffect(() => {
    const el = barRef.current;
    if (!el) return;
    const update = (width: number) => {
      if (width > 0) setBarWidth(width);
    };
    update(el.getBoundingClientRect().width);
    const observer = new ResizeObserver((entries) => {
      update(entries[0].contentRect.width);
    });
    observer.observe(el);
    return () => {
      observer.disconnect();
    };
  }, []);

  // Keep the playhead on screen while zoomed, but never fight an active
  // gesture. A plain tap-seek has no gesture ref, so it still reveals the
  // playhead; panning/zooming the view intentionally is left alone.
  useEffect(() => {
    if (!(duration > 0)) return;
    if (dragRef.current || panRef.current || pinchRef.current) return;
    setView((v) => {
      const span = v.end - v.start;
      if (!(span > 0) || span >= duration - 1e-6) return v;
      const margin = span * 0.08;
      if (currentTime >= v.start + margin && currentTime <= v.end - margin) {
        return v;
      }
      const start = clamp(currentTime - span / 2, 0, duration - span);
      return sameWindow(v, { start, end: start + span })
        ? v
        : { start, end: start + span };
    });
  }, [currentTime, duration]);

  // Reveal a selection that sits entirely outside the window (e.g. an event
  // tagged at the playhead). Never runs mid-gesture, and clicking a visible
  // band leaves the view untouched.
  useEffect(() => {
    if (!(duration > 0) || selectedIndex === null) return;
    if (dragRef.current || panRef.current || pinchRef.current) return;
    const span = events[selectedIndex];
    setView((v) => {
      const win = v.end - v.start;
      if (!(win > 0)) return v;
      if (span[1] >= v.start && span[0] <= v.end) return v;
      const mid = (span[0] + span[1]) / 2;
      const start = clamp(mid - win / 2, 0, duration - win);
      return sameWindow(v, { start, end: start + win })
        ? v
        : { start, end: start + win };
    });
  }, [duration, events, selectedIndex]);

  // Desktop wheel: ⌘/Ctrl zooms about the cursor, otherwise scroll pans. Native
  // (non-passive) so we can stop the browser's own ctrl+wheel page zoom.
  useEffect(() => {
    const el = barRef.current;
    if (!el) return;
    const handleWheel = (e: WheelEvent) => {
      if (!(duration > 0)) return;
      const rect = el.getBoundingClientRect();
      if (!(rect.width > 0)) return;
      e.preventDefault();
      if (e.ctrlKey || e.metaKey) {
        const fraction = clamp((e.clientX - rect.left) / rect.width, 0, 1);
        setView((v) => {
          const span = v.end - v.start;
          if (!(span > 0)) return v;
          const anchor = v.start + fraction * span;
          return zoomWindowAt(v, duration, anchor, Math.exp(e.deltaY * 0.0025));
        });
      } else {
        const delta = e.deltaX !== 0 ? e.deltaX : e.deltaY;
        setView((v) => panWindow(v, duration, (delta / rect.width) * (v.end - v.start)));
      }
    };
    el.addEventListener("wheel", handleWheel, { passive: false });
    return () => {
      el.removeEventListener("wheel", handleWheel);
    };
  }, [duration]);

  const rulerTicks = useMemo(
    () => buildRulerTicks(view.start, view.end),
    [view.start, view.end]
  );

  const pxPerSec = windowSpan > 0 && barWidth > 0 ? barWidth / windowSpan : 0;
  const secondTicks = useMemo(
    () =>
      pxPerSec >= MINOR_TICK_MIN_PX_PER_SEC
        ? buildSecondTicks(view.start, view.end)
        : [],
    [pxPerSec, view.start, view.end]
  );

  /* -------------------- zoom controls -------------------- */

  const canZoomIn = duration > 0 && windowSpan > minSpan + 1e-6;
  const canZoomOut = duration > 0 && windowSpan < duration - 1e-6;
  const canZoomToSelection = duration > 0 && selectedIndex !== null;

  const zoomBy = useCallback(
    (spanFactor: number) => {
      if (!(duration > 0)) return;
      const anchor =
        currentTime >= view.start && currentTime <= view.end
          ? currentTime
          : (view.start + view.end) / 2;
      setView((v) => zoomWindowAt(v, duration, anchor, spanFactor));
    },
    [currentTime, duration, view.start, view.end]
  );

  const fitView = useCallback(() => {
    if (!(duration > 0)) return;
    setView({ start: 0, end: duration });
  }, [duration]);

  const zoomToSelection = useCallback(() => {
    if (!(duration > 0) || selectedIndex === null) return;
    const [start, end] = events[selectedIndex];
    const length = Math.max(end - start, MIN_EVENT_SECONDS);
    const padding = Math.max(length * 0.5, 1);
    const nextSpan = clamp(length + padding * 2, minWindowSpan(duration), duration);
    const nextStart = clamp(
      (start + end) / 2 - nextSpan / 2,
      0,
      duration - nextSpan
    );
    setView({ start: nextStart, end: nextStart + nextSpan });
  }, [duration, events, selectedIndex]);

  /* -------------------- gesture teardown -------------------- */

  /** Drop a half-finished single-pointer gesture before a pinch takes over. */
  const abortSingleGesture = useCallback(() => {
    panRef.current = null;
    const drag = dragRef.current;
    if (!drag) return;
    dragRef.current = null;
    // Roll the preview back so the aborted drag leaves no undo step behind.
    onPreviewChange(drag.index, drag.origin);
    onEndChange();
  }, [onEndChange, onPreviewChange]);

  /* -------------------- pointer handling -------------------- */

  const beginBandDrag = (
    e: ReactPointerEvent<HTMLDivElement>,
    target: HTMLElement,
    index: number
  ) => {
    const mode = (target.dataset.handle as DragMode | undefined) ?? "move";
    const origin = events[index];
    e.preventDefault();
    e.stopPropagation();
    barRef.current?.setPointerCapture(e.pointerId);
    dragRef.current = {
      pointerId: e.pointerId,
      index,
      mode,
      startX: e.clientX,
      origin: [origin[0], origin[1]],
      stuckTo: null,
    };
    onSelect(index);
    onBeginChange();
  };

  const beginPinch = (rect: DOMRect) => {
    abortSingleGesture();
    const points = [...pointersRef.current.values()].slice(0, 2);
    const dist = Math.hypot(
      points[0].x - points[1].x,
      points[0].y - points[1].y
    );
    const midX = (points[0].x + points[1].x) / 2;
    const span = view.end - view.start;
    const fraction = clamp((midX - rect.left) / rect.width, 0, 1);
    pinchRef.current = {
      startDist: Math.max(dist, 1),
      startSpan: span > 0 ? span : duration,
      anchorTime: view.start + fraction * (span > 0 ? span : duration),
    };
  };

  const handlePointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    const rect = barRef.current?.getBoundingClientRect();
    pointersRef.current.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (pointersRef.current.size >= 2) {
      e.preventDefault();
      // Ignore extra fingers: only the first pair drives the pinch.
      if (!pinchRef.current && duration > 0 && rect) beginPinch(rect);
      return;
    }

    if (!(duration > 0) || !rect) return;

    const target = (e.target as HTMLElement).closest<HTMLElement>(
      "[data-event-index]"
    );
    if (target) {
      beginBandDrag(e, target, Number(target.dataset.eventIndex));
      return;
    }

    const lane = (e.target as HTMLElement).closest<HTMLElement>("[data-lane]")
      ?.dataset.lane;

    if (lane === "overview") {
      e.preventDefault();
      barRef.current?.setPointerCapture(e.pointerId);
      panRef.current = {
        pointerId: e.pointerId,
        kind: "overview",
        startX: e.clientX,
        startStart: view.start,
        secondsPerWidth: duration,
        viewSpan: windowSpan > 0 ? windowSpan : duration,
        moved: false,
        tapTime: clientXToTime(e.clientX, rect),
      };
      return;
    }

    if (lane === "ruler" && e.pointerType === "mouse") {
      // Defer the seek so a horizontal drag can pan; a clean tap still seeks.
      e.preventDefault();
      barRef.current?.setPointerCapture(e.pointerId);
      panRef.current = {
        pointerId: e.pointerId,
        kind: "ruler",
        startX: e.clientX,
        startStart: view.start,
        secondsPerWidth: windowSpan > 0 ? windowSpan : duration,
        viewSpan: windowSpan > 0 ? windowSpan : duration,
        moved: false,
        tapTime: clientXToTime(e.clientX, rect),
      };
      return;
    }

    // Touch on the ruler / empty rail, or mouse on the rail: seek, snapped to
    // the 1s grid like every other edit, and drop the selection.
    onSeek(clamp(Math.round(clientXToTime(e.clientX, rect)), 0, duration));
    onSelect(null);
  };

  const moveBand = (
    e: ReactPointerEvent<HTMLDivElement>,
    rect: DOMRect
  ) => {
    const drag = dragRef.current;
    if (!drag || !(windowSpan > 0)) return;

    const pxToSec = windowSpan / rect.width;
    const breakSec = BREAK_PX * pxToSec;
    const snapSec = MOVE_SNAP_PX * pxToSec;
    // 1s resolution: snap every edge to a whole second (jerky is fine).
    const q = (seconds: number): number => Math.round(seconds);

    const dt = ((e.clientX - drag.startX) / rect.width) * windowSpan;
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

  const movePinch = (rect: DOMRect) => {
    const pinch = pinchRef.current;
    if (!pinch) return;
    const points = [...pointersRef.current.values()].slice(0, 2);
    const dist = Math.hypot(
      points[0].x - points[1].x,
      points[0].y - points[1].y
    );
    const midX = (points[0].x + points[1].x) / 2;
    const fraction = clamp((midX - rect.left) / rect.width, 0, 1);
    const scale = pinch.startDist / Math.max(dist, 1);
    const span = clamp(pinch.startSpan * scale, minSpan, duration);
    const start = clamp(pinch.anchorTime - fraction * span, 0, duration - span);
    setView({ start, end: start + span });
  };

  const movePan = (
    e: ReactPointerEvent<HTMLDivElement>,
    rect: DOMRect
  ) => {
    const pan = panRef.current;
    if (!pan) return;
    const dx = e.clientX - pan.startX;
    if (!pan.moved && Math.abs(dx) > PAN_TAP_PX) pan.moved = true;
    if (!pan.moved) return;
    const deltaSec = (dx / rect.width) * pan.secondsPerWidth;
    // Ruler drag moves the content (window follows the opposite way); overview
    // drag grabs the window itself. Compute absolutely from the gesture start so
    // the accumulated pointer delta is not applied twice.
    const base =
      pan.kind === "ruler"
        ? pan.startStart - deltaSec
        : pan.startStart + deltaSec;
    const start = clamp(base, 0, Math.max(0, duration - pan.viewSpan));
    setView({ start, end: start + pan.viewSpan });
  };

  const handlePointerMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    const point = pointersRef.current.get(e.pointerId);
    if (point) {
      point.x = e.clientX;
      point.y = e.clientY;
    }
    const rect = barRef.current?.getBoundingClientRect();
    if (!rect || !(duration > 0)) return;

    if (pinchRef.current && pointersRef.current.size >= 2) {
      movePinch(rect);
      return;
    }
    if (panRef.current?.pointerId === e.pointerId) {
      movePan(e, rect);
      return;
    }
    if (dragRef.current?.pointerId === e.pointerId) {
      moveBand(e, rect);
    }
  };

  const handlePointerUp = (e: ReactPointerEvent<HTMLDivElement>) => {
    pointersRef.current.delete(e.pointerId);

    if (pinchRef.current) {
      if (pointersRef.current.size < 2) pinchRef.current = null;
      return;
    }

    const pan = panRef.current;
    if (pan?.pointerId === e.pointerId) {
      panRef.current = null;
      const rect = barRef.current?.getBoundingClientRect();
      if (!pan.moved && rect && e.type !== "pointercancel") {
        if (pan.kind === "ruler") {
          onSeek(clamp(Math.round(clientXToTime(e.clientX, rect)), 0, duration));
          onSelect(null);
        } else {
          setView((v) => centerWindowOn(v, duration, pan.tapTime));
        }
      }
      return;
    }

    const drag = dragRef.current;
    if (drag?.pointerId === e.pointerId) {
      dragRef.current = null;
      onEndChange();
    }
  };

  const progressPct = clamp(timeToPct(currentTime), 0, 100);
  const overviewStartPct = duration > 0 ? (view.start / duration) * 100 : 0;
  const overviewEndPct = duration > 0 ? (view.end / duration) * 100 : 100;

  return (
    // No overflow clipping: the resize caps sit slightly outside their band so
    // an event at the window edge keeps a grabbable handle.
    <div
      ref={barRef}
      className="absolute bottom-12 left-0 w-full h-[92px] z-30 touch-none select-none border-t border-white/15 bg-black/60 shadow-[0_-6px_18px_rgba(0,0,0,0.55)]"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
    >
      {/* RULER (top, tap to seek / drag to pan) */}
      <div data-lane="ruler" className="absolute top-0 inset-x-0 h-7">
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

      {/* EVENT RAIL (middle) */}
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

      {/* OVERVIEW (bottom): full clip with the current window highlighted. */}
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

      {/* ZOOM CONTROLS — floated above the ruler on the left so they never
          cover the ruler's time labels on narrow screens. */}
      <div
        className="absolute -top-7 left-1 z-40 flex items-center gap-0.5"
        onPointerDown={(e) => {
          e.stopPropagation();
        }}
      >
        <Button
          variant="ghost"
          size="icon"
          onClick={() => {
            zoomBy(BUTTON_ZOOM);
          }}
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
          onClick={() => {
            zoomBy(1 / BUTTON_ZOOM);
          }}
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
          onClick={fitView}
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
          onClick={zoomToSelection}
          disabled={!canZoomToSelection}
          title="Zoom to selected event"
          aria-label="Zoom to selected event"
          className={`${CONTROL} h-6 w-6 disabled:opacity-30`}
        >
          <ScanSearch className="h-3.5 w-3.5" />
        </Button>
      </div>

      {/* Playhead */}
      {duration > 0 && (
        <div
          className="absolute top-0 bottom-4 w-px z-20 bg-white/90 pointer-events-none"
          style={{ left: `${String(progressPct)}%` }}
        />
      )}
    </div>
  );
}
