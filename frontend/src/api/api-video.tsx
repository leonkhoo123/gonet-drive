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

export const renameFileMoveToDone = async (filePath: string, name: string, angle: number, opId: string = generateOpId()): Promise<void> => {

  await axiosLayer.post(
    "/user/video/rename-done",
    {
      path: filePath,
      newName: name,
      rotateAngle: angle,
      opId
    },
    { headers: { "Content-Type": "application/json" } }
  );
};

/** Event span pair [start, end] in seconds. */
export type VideoEventPair = [number, number];

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
