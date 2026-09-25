import type { VideoProgressBarProps } from "./types";

/**
 * Seek bar with buffered/played fills, detected-event markers and an invisible
 * native range input for dragging. The invisible input's hit area grows while
 * controls are visible so the full bar width stays easy to grab.
 */
export function VideoProgressBar({
  showControls,
  bufferedProgress,
  progress,
  duration,
  currentTime,
  events,
  onScrubStart,
  onScrubEnd,
  onScrub,
}: VideoProgressBarProps) {
  return (
    <div
      className={`absolute w-full transition-all duration-300 bg-gray-700/50 z-30 ${
        showControls ? "bottom-12 h-3" : "bottom-0 h-2"
      }`}
    >
      <div
        className="absolute h-full bg-gray-500 pointer-events-none"
        style={{ width: `${String(bufferedProgress)}%` }}
      />
      <div
        className="absolute h-full bg-white/70 pointer-events-none"
        style={{ width: `${String(progress)}%` }}
      />

      {/* Detected-event markers loaded from .vid_metadata/<video>_timestamps.json.
          A faint emerald tint with a crisp accent line; brighter and glowing
          while the playhead sits inside an event. */}
      {duration > 0 && events.length > 0 && (
        <div className="absolute inset-0 pointer-events-none">
          {events.map(([start, end]) => {
            const left = Math.min(100, Math.max(0, (start / duration) * 100));
            const right = Math.min(100, Math.max(left, (end / duration) * 100));
            const width = Math.max(right - left, 0.4);
            const isInside = currentTime >= start && currentTime <= end;
            return (
              <div
                key={`${String(start)}-${String(end)}`}
                className={`absolute top-0 bottom-0 rounded-[1px] border-b-2 transition-all duration-200 ${
                  isInside
                    ? "bg-emerald-300/35 border-emerald-200 shadow-[0_0_8px_rgba(52,211,153,0.85)]"
                    : "bg-emerald-400/15 border-emerald-400/70"
                }`}
                style={{ left: `${String(left)}%`, width: `${String(width)}%` }}
              />
            );
          })}
        </div>
      )}

      {/* Invisible range input for native dragging/scrubbing */}
      <input
        type="range"
        min={0}
        max={100}
        step="any"
        value={progress || 0}
        onPointerDown={(e) => {
          e.stopPropagation();
          onScrubStart();
        }}
        onPointerUp={(e) => {
          e.stopPropagation();
          onScrubEnd();
        }}
        onPointerCancel={(e) => {
          e.stopPropagation();
          onScrubEnd();
        }}
        onChange={(e) => {
          onScrub(parseFloat(e.target.value));
        }}
        className={`absolute left-0 w-full opacity-0 cursor-pointer m-0 ${
          showControls ? "top-0 h-[60px]" : "inset-0 h-full"
        }`}
      />
    </div>
  );
}
