import type { VideoSurfaceProps } from "./types";

/**
 * Full-screen video surface. Also owns the tap-to-toggle and swipe-to-seek
 * pointer handlers, since they need to sit directly on the video area.
 */
export function VideoSurface({
  videoRef,
  rotation,
  onTap,
  onTouchStart,
  onTouchMove,
  onTouchEnd,
}: VideoSurfaceProps) {
  return (
    <div
      className="flex items-center justify-center h-full touch-none"
      onClick={onTap}
      onTouchStart={onTouchStart}
      onTouchMove={onTouchMove}
      onTouchEnd={onTouchEnd}
      onTouchCancel={onTouchEnd}
    >
      <video
        ref={videoRef}
        autoPlay
        playsInline
        preload="auto"
        className="w-full h-full object-contain"
        style={{ transform: `rotate(${String(rotation)}deg)` }}
      />
    </div>
  );
}
