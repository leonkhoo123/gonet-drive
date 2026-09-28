import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { MIN_EVENT_SECONDS } from "@/hooks/useVideoPlayerV2/useVideoEventEditor";
import {
  EventRail,
  TimelineOverview,
  TimelinePlayhead,
  TimelineRuler,
  TimelineZoomControls,
} from "./EventTimelineLayers";
import {
  buildRulerTicks,
  buildSecondTicks,
  clamp,
  minWindowSpan,
  normalizeWindow,
  panWindow,
  zoomWindowAt,
  type ViewWindow,
} from "./timelineViewport";
import type { EventEditorBarProps } from "./types";
import { useTimelineGestures } from "./useTimelineGestures";

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

  const { handlePointerDown, handlePointerMove, handlePointerUp, isGestureActive } =
    useTimelineGestures({
      barRef,
      events,
      duration,
      onSelect,
      onSeek,
      onBeginChange,
      onPreviewChange,
      onEndChange,
      view,
      setView,
      windowSpan,
      minSpan,
      clientXToTime,
    });

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
    if (isGestureActive()) return;
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
  }, [currentTime, duration, isGestureActive]);

  // Reveal a selection that sits entirely outside the window (e.g. an event
  // tagged at the playhead). Never runs mid-gesture, and clicking a visible
  // band leaves the view untouched.
  useEffect(() => {
    if (!(duration > 0) || selectedIndex === null) return;
    if (isGestureActive()) return;
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
  }, [duration, events, selectedIndex, isGestureActive]);

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
      <TimelineRuler
        duration={duration}
        progressPct={progressPct}
        canZoomOut={canZoomOut}
        rulerTicks={rulerTicks}
        secondTicks={secondTicks}
        timeToPct={timeToPct}
      />

      {/* EVENT RAIL (middle) */}
      <EventRail
        events={events}
        duration={duration}
        view={view}
        selectedIndex={selectedIndex}
        timeToPct={timeToPct}
      />

      {/* OVERVIEW (bottom): full clip with the current window highlighted. */}
      <TimelineOverview
        events={events}
        duration={duration}
        overviewStartPct={overviewStartPct}
        overviewEndPct={overviewEndPct}
      />

      {/* ZOOM CONTROLS — floated above the ruler on the left so they never
          cover the ruler's time labels on narrow screens. */}
      <TimelineZoomControls
        canZoomOut={canZoomOut}
        canZoomIn={canZoomIn}
        canZoomToSelection={canZoomToSelection}
        onZoomOut={() => {
          zoomBy(BUTTON_ZOOM);
        }}
        onZoomIn={() => {
          zoomBy(1 / BUTTON_ZOOM);
        }}
        onFit={fitView}
        onZoomToSelection={zoomToSelection}
      />

      {/* Playhead */}
      <TimelinePlayhead duration={duration} progressPct={progressPct} />
    </div>
  );
}
