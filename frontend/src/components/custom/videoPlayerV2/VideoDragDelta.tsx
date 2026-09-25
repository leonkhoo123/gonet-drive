import { formatTime } from "@/utils/videoPlayerV2Events";
import type { VideoDragDeltaProps } from "./types";

/** Floating "+/-m:ss" popup shown while swipe-seeking. */
export function VideoDragDelta({
  seconds,
}: VideoDragDeltaProps) {
  return (
    <div className="absolute top-12 left-1/2 -translate-x-1/2 bg-black/60 text-white px-4 py-2 rounded-full font-bold text-lg tracking-wider pointer-events-none z-20">
      {seconds >= 0 ? "+" : "-"}
      {formatTime(Math.abs(seconds))}
    </div>
  );
}
