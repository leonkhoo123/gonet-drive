/**
 * Pure geometry for the zoomable edit timeline. The editor shows a sub-range of
 * the clip, `[start, end]`, instead of the whole `[0, duration]`; everything in
 * `EventEditorBar` maps time ↔ x through this window. Keeping the maths here
 * (no React, no DOM) makes the clamping rules easy to reason about and to test.
 */

/** Maximum magnification: the window never gets smaller than duration / 64. */
export const MAX_ZOOM = 64;

/** A visible sub-range of the clip, in absolute source seconds. */
export interface ViewWindow {
  start: number;
  end: number;
}

export const clamp = (value: number, lo: number, hi: number): number =>
  Math.min(hi, Math.max(lo, value));

/** Smallest window we allow for a given clip (the max-zoom floor). */
export const minWindowSpan = (duration: number): number =>
  duration > 0 ? duration / MAX_ZOOM : 0;

/** Force a window back inside the clip and inside the allowed zoom range. */
export function normalizeWindow(view: ViewWindow, duration: number): ViewWindow {
  if (!(duration > 0)) return { start: 0, end: 0 };
  let span = view.end - view.start;
  // A degenerate window (e.g. duration arrived after mount) means "fit".
  if (!(span > 0) || !Number.isFinite(span)) span = duration;
  span = clamp(span, minWindowSpan(duration), duration);
  const start = clamp(view.start, 0, duration - span);
  return { start, end: start + span };
}

/**
 * Scale the window by `spanFactor` about `anchorTime` (kept under the same x).
 * `spanFactor < 1` zooms in, `> 1` zooms out.
 */
export function zoomWindowAt(
  view: ViewWindow,
  duration: number,
  anchorTime: number,
  spanFactor: number
): ViewWindow {
  if (!(duration > 0)) return view;
  const span = view.end - view.start;
  if (!(span > 0)) return view;
  const nextSpan = clamp(span * spanFactor, minWindowSpan(duration), duration);
  const fraction = clamp((anchorTime - view.start) / span, 0, 1);
  const start = clamp(anchorTime - fraction * nextSpan, 0, duration - nextSpan);
  return { start, end: start + nextSpan };
}

/** Slide the window by `deltaSeconds` (positive moves it later in the clip). */
export function panWindow(
  view: ViewWindow,
  duration: number,
  deltaSeconds: number
): ViewWindow {
  const span = view.end - view.start;
  if (!(duration > 0) || !(span > 0)) return view;
  const start = clamp(view.start + deltaSeconds, 0, duration - span);
  return { start, end: start + span };
}

/** Recentre the window on `time`, preserving the current zoom level. */
export function centerWindowOn(
  view: ViewWindow,
  duration: number,
  time: number
): ViewWindow {
  if (!(duration > 0)) return view;
  const current = view.end - view.start;
  const span = clamp(
    current > 0 ? current : duration,
    minWindowSpan(duration),
    duration
  );
  const start = clamp(time - span / 2, 0, duration - span);
  return { start, end: start + span };
}

/** Round intervals (seconds) the ruler labels prefer, so times read formally. */
const RULER_STEPS = [1, 2, 5, 10, 15, 20, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 10800, 21600];
/** Aim for roughly this many labels; the step snaps up to the next nice value. */
const RULER_TARGET_LABELS = 6;

export type TickAlign = "left" | "center" | "right";

export interface RulerTick {
  t: number;
  align: TickAlign;
}

/** Nice labelled interval (seconds) for a window of the given length. */
export function rulerStep(span: number): number {
  const raw = span / RULER_TARGET_LABELS;
  return RULER_STEPS.find((s) => s >= raw) ?? Math.ceil(raw);
}

/**
 * Labelled ticks that fall inside `[start, end]`, on whole multiples of the
 * chosen step so the times stay round while panning/zooming. Edge ticks align
 * inward so their labels do not spill past the bar.
 */
export function buildRulerTicks(start: number, end: number): RulerTick[] {
  const span = end - start;
  if (!(span > 0)) return [];
  const step = rulerStep(span);
  const first = Math.ceil(start / step) * step;
  const ticks: RulerTick[] = [];
  for (let t = first; t <= end + 1e-6; t += step) {
    const rel = (t - start) / span;
    const align: TickAlign = rel < 0.02 ? "left" : rel > 0.98 ? "right" : "center";
    ticks.push({ t, align });
  }
  return ticks;
}

/**
 * Whole-second gridlines inside the window. Callers only render these once the
 * zoom is deep enough that a second is a sane number of pixels apart.
 */
export function buildSecondTicks(start: number, end: number): number[] {
  const ticks: number[] = [];
  for (let t = Math.ceil(start); t <= end + 1e-6; t += 1) ticks.push(t);
  return ticks;
}
