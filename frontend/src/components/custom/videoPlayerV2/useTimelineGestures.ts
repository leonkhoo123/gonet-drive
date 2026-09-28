import {
  useCallback,
  useRef,
  type Dispatch,
  type PointerEvent as ReactPointerEvent,
  type RefObject,
  type SetStateAction,
} from "react";
import { MIN_EVENT_SECONDS } from "@/hooks/useVideoPlayerV2/useVideoEventEditor";
import type { EventSpan } from "@/utils/videoPlayerV2Events";
import { centerWindowOn, clamp, type ViewWindow } from "./timelineViewport";

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

export interface UseTimelineGesturesArgs {
  barRef: RefObject<HTMLDivElement | null>;
  events: EventSpan[];
  duration: number;
  onSelect: (index: number | null) => void;
  onSeek: (seconds: number) => void;
  onBeginChange: () => void;
  onPreviewChange: (index: number, span: EventSpan) => void;
  onEndChange: () => void;
  /** Visible window, mirrored from the editor's state. */
  view: ViewWindow;
  setView: Dispatch<SetStateAction<ViewWindow>>;
  /** Span of the visible window (0 when degenerate). */
  windowSpan: number;
  /** Minimum allowed window span for the current clip. */
  minSpan: number;
  /** Pointer/client-x → absolute clip seconds for the current window. */
  clientXToTime: (clientX: number, rect: DOMRect) => number;
}

/**
 * All pointer-gesture handling for the edit timeline: band move/resize, ruler
 * and overview pans, and two-finger pinch zoom. State lives in refs so a
 * gesture never re-renders mid-flight; the handlers are re-created per render
 * like the JSX callbacks that use them.
 */
export function useTimelineGestures({
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
}: UseTimelineGesturesArgs) {
  const dragRef = useRef<DragState | null>(null);
  const panRef = useRef<PanState | null>(null);
  const pinchRef = useRef<PinchState | null>(null);
  const pointersRef = useRef<Map<number, { x: number; y: number }>>(new Map());

  /** True while a drag/pan/pinch is in flight, so view effects can stand down. */
  const isGestureActive = useCallback(
    () =>
      dragRef.current !== null ||
      panRef.current !== null ||
      pinchRef.current !== null,
    []
  );

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

    if (lane === "ruler") {
      // Defer the seek so a horizontal drag can pan the zoomed window (content
      // follows the finger); a clean tap still seeks. Works for mouse and touch
      // alike, so the timestamp lane is a pan handle just like the overview.
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

    // Touch/pen on the empty rail, or mouse on the rail: seek, snapped to the
    // 1s grid like every other edit, and drop the selection.
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

  return { handlePointerDown, handlePointerMove, handlePointerUp, isGestureActive };
}
