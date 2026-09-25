import { formatTime } from "@/utils/videoPlayerV2Events";
import type { VideoTimeDisplayProps } from "./types";

/** Bottom-left playback time, remaining time and status flags. */
export function VideoTimeDisplay({
  currentTime,
  duration,
  disqualified,
  isNewName,
  newName,
  fileName,
  isRotation,
  rotation,
}: VideoTimeDisplayProps) {
  return (
    <div className="absolute bottom-2 left-1 w-full text-left text-white text-sm select-none z-10">
      <span>
        {formatTime(currentTime)} / {formatTime(duration)}
      </span>
      <span className="text-gray-400 ml-2">
        (-{formatTime(duration - currentTime)})
      </span>

      {disqualified ? (
        <span className="ml-3 text-red-500 font-bold">Disqualified</span>
      ) : (
        ""
      )}
      {isNewName ? (
        <span className="ml-3 text-green-500 font-bold">{newName}</span>
      ) : (
        <span className="ml-3 text-white/40 font-bold">{fileName}</span>
      )}
      {isRotation ? (
        <span className="ml-3 text-green-500 font-bold">({rotation}°)</span>
      ) : (
        ""
      )}
    </div>
  );
}
