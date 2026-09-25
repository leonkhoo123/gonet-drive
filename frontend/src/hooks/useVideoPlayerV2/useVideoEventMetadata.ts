import { useEffect, useState } from "react";
import axiosLayer from "@/api/axiosLayer";
import {
  METADATA_DIRNAME,
  parseEventSpans,
  type EventSpan,
} from "@/utils/videoPlayerV2Events";

/**
 * Load the sibling metadata JSON written by the AI pipeline:
 * `<video folder>/.vid_metadata/<video filename>_timestamps.json`
 * (e.g. `clip.mp4_timestamps.json` - the extension is kept in the name).
 *
 * A missing or malformed file is non-fatal - the player just shows no markers.
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

    const slash = filePath.lastIndexOf("/");
    const dir = slash >= 0 ? filePath.slice(0, slash) : "";
    const metaPath = `${dir}/${METADATA_DIRNAME}/${fileName}_timestamps.json`;

    let cancelled = false;
    setEvents([]);

    axiosLayer
      .get<unknown>(`/user/document/read/file${encodeURI(metaPath)}`)
      .then((res) => {
        if (cancelled) return;
        const payload: unknown =
          typeof res.data === "string"
            ? (JSON.parse(res.data) as unknown)
            : res.data;
        setEvents(parseEventSpans(payload));
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
