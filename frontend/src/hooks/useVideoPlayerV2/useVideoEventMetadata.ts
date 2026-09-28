import { useEffect, useState } from "react";
import { getVideoEvents } from "@/api/api-video";
import { parseEventSpans, type EventSpan } from "@/utils/videoPlayerV2Events";

/** Where the event metadata lives for a video. */
export type VideoEventSource = "embedded" | "sidecar";

export interface VideoEventMetadata {
  events: EventSpan[];
  /** `null` when the video has no event metadata at all (backend 404). */
  source: VideoEventSource | null;
}

/**
 * Load the detected event spans for a video from the backend.
 *
 * The backend prefers the metadata embedded in the container and falls back to
 * the sibling `<video folder>/.vid_metadata/<video filename>_timestamps.json`
 * sidecar. A 404 (no metadata anywhere) is non-fatal - the player just shows
 * no markers and reports a null source.
 */
export function useVideoEventMetadata(
  isOpen: boolean,
  filePath: string,
  fileName: string,
): VideoEventMetadata {
  const [data, setData] = useState<VideoEventMetadata>({
    events: [],
    source: null,
  });

  useEffect(() => {
    if (!isOpen || !filePath) {
      setData({ events: [], source: null });
      return;
    }

    let cancelled = false;
    setData({ events: [], source: null });

    getVideoEvents(filePath)
      .then((res) => {
        if (cancelled) return;
        setData({ events: parseEventSpans(res.events), source: res.source });
      })
      .catch(() => {
        if (!cancelled) setData({ events: [], source: null });
      });

    return () => {
      cancelled = true;
    };
  }, [isOpen, filePath, fileName]);

  return data;
}
