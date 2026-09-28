import axiosLayer from './axiosLayer';   // axios instance WITHOUT token
import { unwrap, type ApiEnvelope } from './envelope';
import { generateOpId } from "../utils/id";


export const postDisqualified = async (filePath: string, opId: string = generateOpId()): Promise<void> => {
  await axiosLayer.post(
    "/user/video/disqualified",
    { path: filePath, opId }, // request body
    {
      headers: { "Content-Type": "application/json" },
    }
  );
};

/** Event span pair [start, end] in seconds. */
export type VideoEventPair = [number, number];

export const renameFileMoveToDone = async (
  filePath: string,
  name: string,
  angle: number,
  events?: VideoEventPair[],
  opId: string = generateOpId()
): Promise<void> => {
  await axiosLayer.post(
    "/user/video/rename-done",
    {
      path: filePath,
      newName: name,
      rotateAngle: angle,
      // Omit the field entirely for a plain rename so the backend keeps its
      // sidecar-driven behaviour; an empty array intentionally clears events.
      ...(events !== undefined ? { events } : {}),
      opId
    },
    { headers: { "Content-Type": "application/json" } }
  );
};

export interface VideoEventsResponse {
  events: VideoEventPair[];
  /** Where the backend found the events. */
  source: "embedded" | "sidecar";
}

/**
 * Fetch the detected event spans for a video. The backend checks the container
 * tag first, then falls back to the sidecar JSON, and returns 404 when neither
 * exists (the caller treats that as "no events").
 */
export const getVideoEvents = async (filePath: string): Promise<VideoEventsResponse> => {
  const response = await axiosLayer.get<ApiEnvelope<VideoEventsResponse>>(
    `/user/video/metadata/file${encodeURI(filePath)}`
  );
  return unwrap(response);
};

export interface VideoMetadataCommitResponse {
  message: string;
  /** Operation id to track the async embed/sidecar job over the WebSocket. */
  opId: string;
}

/**
 * Persist edited event highlights. The backend rewrites the sidecar and, for
 * MP4-family files, re-embeds the container tags in place as an async job that
 * reports progress over the WebSocket (same channel as copy/move/rename-done).
 */
export const commitVideoEvents = async (
  filePath: string,
  events: VideoEventPair[],
  opId: string = generateOpId()
): Promise<VideoMetadataCommitResponse> => {
  const response = await axiosLayer.post<ApiEnvelope<VideoMetadataCommitResponse>>(
    "/user/video/metadata/commit",
    { path: filePath, events, opId },
    { headers: { "Content-Type": "application/json" } }
  );
  return unwrap(response);
};

export interface VideoDurationResponse {
  /** Absolute source duration in seconds (0 when unknown). */
  duration: number;
  width?: number;
  height?: number;
}

/**
 * Fetch the absolute duration of a video. Needed by the compressed player
 * because every transcoded segment starts at 0, so the browser can never learn
 * the real total from the media element.
 */
export const getVideoDuration = async (
  filePath: string
): Promise<VideoDurationResponse> => {
  const response = await axiosLayer.get<ApiEnvelope<VideoDurationResponse>>(
    `/user/video/duration/file${encodeURI(filePath)}`
  );
  return unwrap(response);
};
