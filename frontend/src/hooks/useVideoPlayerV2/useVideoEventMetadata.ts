import { useEffect, useState } from "react";
import { getVideoEvents } from "@/api/api-video";
import { parseEventSpans, type EventSpan } from "@/utils/videoPlayerV2Events";

/**
 * Load the detected event spans for a video from the backend.
 *
 * The backend prefers the metadata embedded in the container and falls back to
 * the sibling `<video folder>/.vid_metadata/<video filename>_timestamps.json`
 * sidecar. A 404 (no metadata anywhere) is non-fatal - the player just shows
 * no markers.
 */
export function useVideoEventMetadata(
  isOpen: boolean,
  filePath: string,
  fileName: string,
): EventSpan[] {
  const [events, setEvents] = useState<EventSpan[]>([]);

  useEffect(() => {
    if (!isOpen || !filePath) {
      setEvents([]);
      return;
    }

    let cancelled = false;
    setEvents([]);

    getVideoEvents(filePath)
      .then((res) => {
        if (cancelled) return;
        setEvents(parseEventSpans(res.events));
      })
      .catch(() => {
        if (!cancelled) setEvents([]);
      });

    return () => {
      cancelled = true;
    };
  }, [isOpen, filePath, fileName]);

  return events;
}
