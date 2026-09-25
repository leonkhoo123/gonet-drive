/** A detected event as an absolute [start, end] pair in seconds. */
export type EventSpan = [number, number];

/** Directory (relative to the video) holding the AI-generated timestamps JSON. */
export const METADATA_DIRNAME = ".vid_metadata";

/**
 * Read the event array out of a timestamps JSON payload.
 * Mirrors the tolerant parsing of the Python player: accepts
 * `{"events": [...]}`, the compact `{"e": [...]}`, or a bare array.
 */
const extractEventArray = (data: unknown): unknown[] => {
  if (Array.isArray(data)) return data;
  if (data && typeof data === "object") {
    const obj = data as Record<string, unknown>;
    if (Array.isArray(obj.events)) return obj.events;
    if (Array.isArray(obj.e)) return obj.e;
  }
  return [];
};

/** Normalise raw JSON entries into sorted, clamped [start, end] spans. */
export const parseEventSpans = (data: unknown): EventSpan[] => {
  const spans: EventSpan[] = [];

  for (const item of extractEventArray(data)) {
    let start: number;
    let end: number;

    if (Array.isArray(item) && item.length >= 2) {
      start = Number(item[0]);
      end = Number(item[1]);
    } else if (item && typeof item === "object") {
      const obj = item as Record<string, unknown>;
      start = Number(obj.start);
      end = Number(obj.end);
    } else {
      continue;
    }

    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    spans.push([Math.min(start, end), Math.max(start, end)]);
  }

  spans.sort((a, b) => a[0] - b[0]);
  return spans;
};

/** Format a number of seconds as `m:ss`. */
export const formatTime = (t: number): string =>
  `${String(Math.floor(t / 60))}:${String(Math.floor(t % 60)).padStart(2, "0")}`;
