import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { commitVideoEvents, renameFileMoveToDone } from "@/api/api-video";
import { useVideoEventMetadata, type VideoEventSource } from "./useVideoEventMetadata";
import type { EventSpan } from "@/utils/videoPlayerV2Events";

/**
 * Shortest event we allow in the editor. Every edit-mode handle enforces this:
 * resize caps cannot shrink a band below it, the overlap trim drops anything
 * that collapses past it, and nudges clamp to it.
 */
export const MIN_EVENT_SECONDS = 1;
/** Default length of a newly tagged event. */
const NEW_EVENT_SECONDS = 3;
/** Undo depth ceiling. */
const HISTORY_LIMIT = 50;

interface UseVideoEventEditorParams {
  isOpen: boolean;
  filePath: string;
  fileName: string;
  /** Absolute source duration in seconds (0 when unknown). */
  duration: number;
  /** Current playhead in absolute source seconds. */
  currentTime: number;
}

/**
 * A draft event with a stable identity. Selection is tracked by id (not array
 * index) so it survives the sorts/drops that overlap-trimming performs.
 */
interface EditableEvent {
  id: number;
  span: EventSpan;
}

const clamp = (value: number, lo: number, hi: number): number =>
  Math.min(hi, Math.max(lo, value));

const cloneSpan = ([start, end]: EventSpan): EventSpan => [start, end];

const cloneItems = (items: EditableEvent[]): EditableEvent[] =>
  items.map((item) => ({ id: item.id, span: cloneSpan(item.span) }));

const sameSpans = (a: EventSpan[], b: EventSpan[]): boolean =>
  a.length === b.length &&
  a.every((span, i) => span[0] === b[i][0] && span[1] === b[i][1]);

const sameItems = (a: EditableEvent[], b: EditableEvent[]): boolean =>
  a.length === b.length &&
  a.every((item, i) => item.span[0] === b[i].span[0] && item.span[1] === b[i].span[1]);

/**
 * Owns the event-highlight editing session for the compress player: a
 * client-only draft, single selection, undo/redo, and the async commit.
 *
 * Overlap policy (the "polish"): starts are authoritative because skipping and
 * emphasis happen at the start of an event. So on entering, after every edit
 * and before saving, the draft is normalized by clamping each event's end to
 * the next event's start and dropping anything that collapses:
 *
 *   A[1,10], B[9,13]  →  A[1,9], B[9,13]
 *
 * The backend stores whatever it is given and does not re-check overlaps.
 */
export function useVideoEventEditor({
  isOpen,
  filePath,
  fileName,
  duration,
  currentTime,
}: UseVideoEventEditorParams) {
  const loaded = useVideoEventMetadata(isOpen, filePath, fileName);

  // Locally applied save result, until the metadata is re-fetched on next open.
  const [committed, setCommitted] = useState<EventSpan[] | null>(null);
  // Where the saved events live. Rule B: a container that already had an
  // embedded tag is re-embedded; anything else is written sidecar-only.
  const [committedSource, setCommittedSource] = useState<VideoEventSource | null>(null);
  const events = committed ?? loaded.events;
  const metadataSource = committedSource ?? loaded.source;

  const [items, setItems] = useState<EditableEvent[]>([]);
  const [baseline, setBaseline] = useState<EventSpan[]>([]);
  const [isEditing, setIsEditing] = useState(false);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [past, setPast] = useState<EditableEvent[][]>([]);
  const [future, setFuture] = useState<EditableEvent[][]>([]);
  const [isSaving, setIsSaving] = useState(false);

  const nextIdRef = useRef(1);
  const itemsRef = useRef<EditableEvent[]>([]);
  const interactionSnapshot = useRef<EditableEvent[] | null>(null);

  const draft = useMemo(() => items.map((item) => item.span), [items]);

  const selectedIndex = useMemo(() => {
    if (selectedId === null) return null;
    const index = items.findIndex((item) => item.id === selectedId);
    return index === -1 ? null : index;
  }, [items, selectedId]);

  const selectedSpan =
    selectedIndex === null ? null : (items[selectedIndex]?.span ?? null);

  // A clip change resets everything: no committed override, no edit session.
  useEffect(() => {
    setCommitted(null);
    setCommittedSource(null);
    setIsEditing(false);
    setSelectedId(null);
    setPast([]);
    setFuture([]);
  }, [filePath]);

  useEffect(() => {
    if (!isOpen) {
      setIsEditing(false);
      setSelectedId(null);
    }
  }, [isOpen]);

  const makeItem = useCallback((span: EventSpan): EditableEvent => {
    return { id: nextIdRef.current++, span: cloneSpan(span) };
  }, []);

  const clampSpan = useCallback(
    ([start, end]: EventSpan): EventSpan => {
      const max = duration > 0 ? duration : Number.POSITIVE_INFINITY;
      let s = clamp(start, 0, max);
      let e = clamp(end, 0, max);
      if (e < s) [s, e] = [e, s];
      if (e - s < MIN_EVENT_SECONDS) {
        e = Math.min(max, s + MIN_EVENT_SECONDS);
        s = Math.max(0, e - MIN_EVENT_SECONDS);
      }
      return [s, e];
    },
    [duration]
  );

  /** Enforce the no-overlap policy: clamp ends, drop collapsed spans, sort. */
  const trimItems = useCallback(
    (source: EditableEvent[]): EditableEvent[] => {
      const sorted = [...source].sort((a, b) => a.span[0] - b.span[0]);
      const out: EditableEvent[] = [];
      for (let i = 0; i < sorted.length; i++) {
        const item = sorted[i];
        let start = item.span[0];
        let end = item.span[1];
        if (duration > 0) {
          start = clamp(start, 0, duration);
          end = clamp(end, 0, duration);
        }
        const nextStart =
          i + 1 < sorted.length ? sorted[i + 1].span[0] : null;
        if (nextStart !== null && nextStart < end) end = nextStart;
        if (end - start >= MIN_EVENT_SECONDS) {
          out.push({ id: item.id, span: [start, end] });
        }
      }
      return out;
    },
    [duration]
  );

  /* -------------------- session -------------------- */

  const enter = useCallback(() => {
    const initial = trimItems(events.map(makeItem));
    itemsRef.current = initial;
    setItems(initial);
    setBaseline(initial.map((item) => cloneSpan(item.span)));
    setSelectedId(null);
    setPast([]);
    setFuture([]);
    setIsEditing(true);
  }, [events, makeItem, trimItems]);

  const cancel = useCallback(() => {
    if (isSaving) return;
    setIsEditing(false);
    setSelectedId(null);
  }, [isSaving]);

  const save = useCallback(
    async (options?: { newName?: string; rotation?: number }): Promise<boolean> => {
      if (isSaving) return false;
      // Final polish so the payload is overlap-free regardless of how the draft
      // was left; the backend does not check this.
      const polished = trimItems(itemsRef.current).map((item) => cloneSpan(item.span));
      setIsSaving(true);
      try {
        if (options?.newName) {
          // Terminal save: events, rotation and the rename ship together in one
          // job, so the file moves to done/ only after the events are written.
          await renameFileMoveToDone(
            filePath,
            options.newName,
            options.rotation ?? 0,
            polished
          );
          toast.success("Saving events and moving to Done…");
        } else {
          await commitVideoEvents(filePath, polished);
          toast.success("Saving events…");
        }
        setCommitted(polished);
        setCommittedSource(loaded.source === "embedded" ? "embedded" : "sidecar");
        setIsEditing(false);
        setSelectedId(null);
        return true;
      } catch {
        toast.error(options?.newName ? "Could not save and rename" : "Could not save events");
        return false;
      } finally {
        setIsSaving(false);
      }
    },
    [filePath, isSaving, loaded.source, trimItems]
  );

  /** Apply a transition, pushing one undo step. `selectId` re-targets selection. */
  const commit = useCallback(
    (next: EditableEvent[], selectId?: number | null) => {
      const prev = itemsRef.current;
      const trimmed = trimItems(next);
      if (selectId !== undefined) setSelectedId(selectId);
      if (sameItems(prev, trimmed)) return;
      setPast((p) => [...p.slice(-(HISTORY_LIMIT - 1)), cloneItems(prev)]);
      setFuture([]);
      itemsRef.current = trimmed;
      setItems(trimmed);
    },
    [trimItems]
  );

  /* -------------------- selection -------------------- */

  const select = useCallback((index: number | null) => {
    setSelectedId(index === null ? null : (itemsRef.current[index]?.id ?? null));
  }, []);

  /* -------------------- discrete edits (each = one undo step) -------------------- */

  const addEvent = useCallback(
    (start: number, end: number) => {
      const item = makeItem(clampSpan([start, end]));
      commit([...itemsRef.current, item], item.id);
    },
    [clampSpan, commit, makeItem]
  );

  /** True when `point` is where one event ends and the next begins. */
  const isStickingPoint = useCallback((point: number): boolean => {
    const items = itemsRef.current;
    return (
      items.some((item) => item.span[1] === point) &&
      items.some((item) => item.span[0] === point)
    );
  }, []);

  /**
   * Tag a new event at the playhead. When the playhead sits inside an existing
   * event A, that event is *split* rather than overwritten:
   *
   *   A[1,10], playhead 5, default 3  →  A[1,5]  B[5,8]  C[8,10]
   *
   * The left part is closed at the split and the tail (from the new event's end
   * back to A's original end) is kept as a new event. The split is snapped to
   * the 1s grid and nudged right, if needed, so the left keeps its 1s floor.
   */
  const addAtPlayhead = useCallback(() => {
    const max = duration > 0 ? duration : currentTime + NEW_EVENT_SECONDS;
    const upper = Math.max(0, max - MIN_EVENT_SECONDS);
    const play = clamp(Math.round(currentTime), 0, upper);

    // The shared edge where two events touch is ambiguous to split: whichever
    // neighbour is treated as the host, the trim can collapse the other one.
    // Refuse to create there rather than silently dropping an event.
    if (isStickingPoint(play)) {
      toast.info("Can't add an event on an event boundary");
      return;
    }

    let split = play;
    const host = itemsRef.current.find(
      (item) => item.span[0] <= play && item.span[1] > play
    );
    if (host) {
      const minSplit = Math.ceil(host.span[0] + MIN_EVENT_SECONDS);
      if (split < minSplit) split = minSplit;
    }
    split = clamp(split, 0, upper);

    // Never let the new event spill into a following event: cap its end at the
    // next event's start (the host itself is excluded).
    const nextBoundary = itemsRef.current.reduce<number | null>((acc, item) => {
      if (host && item.id === host.id) return acc;
      if (item.span[0] < split) return acc;
      return acc === null || item.span[0] < acc ? item.span[0] : acc;
    }, null);
    const desiredEnd = Math.min(max, split + NEW_EVENT_SECONDS);
    const newEnd =
      nextBoundary === null ? desiredEnd : Math.min(desiredEnd, nextBoundary);
    const newItem = makeItem([split, newEnd]);

    let next: EditableEvent[];
    if (host) {
      const [hostStart, hostEnd] = host.span;
      const leftovers: EditableEvent[] = [];
      // Left part of the host, closed at the split.
      if (split - hostStart >= MIN_EVENT_SECONDS) {
        leftovers.push({ id: host.id, span: [hostStart, split] });
      }
      // Tail of the host, re-opened after the new event.
      if (hostEnd - newItem.span[1] >= MIN_EVENT_SECONDS) {
        leftovers.push(makeItem([newItem.span[1], hostEnd]));
      }
      next = itemsRef.current.filter((item) => item.id !== host.id).concat(leftovers);
    } else {
      next = itemsRef.current;
    }

    commit([...next, newItem], newItem.id);
  }, [commit, currentTime, duration, isStickingPoint, makeItem]);

  /** Whether the "+ Event" action is allowed at the current playhead. */
  const canAddAtPlayhead = useMemo(() => {
    const max = duration > 0 ? duration : currentTime + NEW_EVENT_SECONDS;
    const upper = Math.max(0, max - MIN_EVENT_SECONDS);
    if (upper <= 0) return false;
    const play = clamp(Math.round(currentTime), 0, upper);
    return !isStickingPoint(play);
  }, [currentTime, duration, isStickingPoint]);

  const deleteSelected = useCallback(() => {
    if (selectedId === null) return;
    commit(
      itemsRef.current.filter((item) => item.id !== selectedId),
      null
    );
  }, [commit, selectedId]);

  /**
   * Discard every edit and restore the event list loaded from the embedded tag
   * or sidecar (the session baseline). Pushed as one undo step so it can itself
   * be undone; a no-op when the draft already matches the baseline.
   */
  const revert = useCallback(() => {
    commit(
      baseline.map((span) => makeItem(span)),
      null
    );
  }, [baseline, commit, makeItem]);

  const setSelectedSpan = useCallback(
    (span: EventSpan) => {
      if (selectedId === null) return;
      commit(
        itemsRef.current.map((item) =>
          item.id === selectedId ? { id: item.id, span: clampSpan(span) } : item
        ),
        selectedId
      );
    },
    [clampSpan, commit, selectedId]
  );

  const readSelectedSpan = useCallback((): EventSpan | null => {
    if (selectedId === null) return null;
    return itemsRef.current.find((item) => item.id === selectedId)?.span ?? null;
  }, [selectedId]);

  const nudgeSelected = useCallback(
    (edge: "start" | "end", delta: number) => {
      const span = readSelectedSpan();
      if (!span) return;
      setSelectedSpan(
        edge === "start" ? [span[0] + delta, span[1]] : [span[0], span[1] + delta]
      );
    },
    [readSelectedSpan, setSelectedSpan]
  );

  const setEdgeToPlayhead = useCallback(
    (edge: "start" | "end") => {
      const span = readSelectedSpan();
      if (!span) return;
      setSelectedSpan(edge === "start" ? [currentTime, span[1]] : [span[0], currentTime]);
    },
    [currentTime, readSelectedSpan, setSelectedSpan]
  );

  /* -------------------- live drag (one undo step for the gesture) -------------------- */

  const beginChange = useCallback(() => {
    interactionSnapshot.current = cloneItems(itemsRef.current);
  }, []);

  const previewChange = useCallback(
    (index: number, span: EventSpan) => {
      // Not trimmed mid-gesture: dropping/sorting during a drag would move the
      // grabbed band and lose the pointer. `endChange` applies the policy.
      const next = itemsRef.current.map((item, i) =>
        i === index ? { id: item.id, span: clampSpan(span) } : item
      );
      itemsRef.current = next;
      setItems(next);
    },
    [clampSpan]
  );

  const endChange = useCallback(() => {
    const snapshot = interactionSnapshot.current;
    interactionSnapshot.current = null;
    const trimmed = trimItems(itemsRef.current);
    if (snapshot && !sameItems(snapshot, trimmed)) {
      setPast((p) => [...p.slice(-(HISTORY_LIMIT - 1)), snapshot]);
      setFuture([]);
    }
    itemsRef.current = trimmed;
    setItems(trimmed);
  }, [trimItems]);

  /* -------------------- history -------------------- */

  const undo = useCallback(() => {
    if (!past.length) return;
    const prev = past[past.length - 1];
    setFuture((f) => [cloneItems(itemsRef.current), ...f].slice(0, HISTORY_LIMIT));
    setPast(past.slice(0, -1));
    itemsRef.current = prev;
    setItems(prev);
    setSelectedId((id) =>
      id !== null && prev.some((item) => item.id === id) ? id : null
    );
  }, [past]);

  const redo = useCallback(() => {
    if (!future.length) return;
    const next = future[0];
    setPast((p) => [...p.slice(-(HISTORY_LIMIT - 1)), cloneItems(itemsRef.current)]);
    setFuture(future.slice(1));
    itemsRef.current = next;
    setItems(next);
    setSelectedId((id) =>
      id !== null && next.some((item) => item.id === id) ? id : null
    );
  }, [future]);

  const isDirty = useMemo(
    () => isEditing && !sameSpans(draft, baseline),
    [isEditing, draft, baseline]
  );

  return {
    events,
    /** Where the current events live: "embedded" or "sidecar" (null if none). */
    metadataSource,
    /** Working copy while editing; mirrors `events` otherwise. */
    draft,
    isEditing,
    isDirty,
    isSaving,
    selectedIndex,
    selectedSpan,
    hasEvents: events.length > 0,
    canUndo: past.length > 0,
    canRedo: future.length > 0,
    canAddAtPlayhead,
    enter,
    cancel,
    save,
    select,
    addEvent,
    addAtPlayhead,
    deleteSelected,
    revert,
    nudgeSelected,
    setEdgeToPlayhead,
    setSelectedSpan,
    beginChange,
    previewChange,
    endChange,
    undo,
    redo,
  };
}

export type VideoEventEditor = ReturnType<typeof useVideoEventEditor>;
