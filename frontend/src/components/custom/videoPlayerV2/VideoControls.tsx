import { Button } from "@/components/ui/button";
import {
  Play,
  Pause,
  Zap,
  SkipBack,
  SkipForward,
  ChevronsLeft,
  ChevronsRight,
  LogOut,
  TextCursorInput,
  ListX,
  RotateCw,
} from "lucide-react";
import type { VideoControlsProps } from "./types";

/** Right-hand vertical control column. */
export function VideoControls({
  showControls,
  isPlaying,
  playbackRate,
  hasEvents,
  onPressStart,
  onPressEnd,
  onSkip,
  onPrevEvent,
  onNextEvent,
  onTogglePlay,
  onChangeSpeed,
  onOpenRename,
  onToggleDisqualified,
  onRotate,
  onClose,
}: VideoControlsProps) {
  return (
    <div
      className={`absolute right-0 top-0 bottom-12 flex flex-col p-1 lg:p-2 transition-opacity duration-300 ${
        showControls ? "opacity-100" : "opacity-0 pointer-events-none"
      }`}
      onMouseDown={onPressStart}
      onMouseUp={onPressEnd}
      onTouchStart={onPressStart}
      onTouchEnd={onPressEnd}
    >
      <div className="m-auto flex flex-col gap-1 lg:gap-2 h-full max-h-[100%] overflow-y-auto scrollbar-hide w-20 lg:w-24 py-2 justify-center">
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(-1);
          }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          <SkipBack className="h-4 w-4 mr-1" /> 1s
        </Button>

        {/* forward 3 sec */}
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(3);
          }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          3s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* forward 1 sec */}
        <Button
          variant="ghost"
          onClick={() => {
            onSkip(1);
          }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          1s <SkipForward className="h-4 w-4 ml-1" />
        </Button>

        {/* previous/next detected event - only rendered when metadata exists */}
        {hasEvents && (
          <>
            <Button
              variant="ghost"
              onClick={onPrevEvent}
              title="Previous event (P or Shift+,)"
              className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
            >
              <ChevronsLeft className="h-4 w-4 mr-1" /> Evt
            </Button>

            <Button
              variant="ghost"
              onClick={onNextEvent}
              title="Next event (N or Shift+.)"
              className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
            >
              Evt <ChevronsRight className="h-4 w-4 ml-1" />
            </Button>
          </>
        )}

        {/* slow mo x0.25 */}
        <Button
          variant="ghost"
          onClick={() => {
            onChangeSpeed(playbackRate !== 1.0 ? 1.0 : 0.25);
          }}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12 px-1"
        >
          <Zap className="h-4 w-4 mr-1" />
          {playbackRate !== 1.0 ? "x1" : "x0.25"}
        </Button>

        {/* play/pause */}
        <div className="text-sm text-white text-center flex items-center justify-center min-h-[20px] shrink-0">
          {playbackRate}x
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={onTogglePlay}
          className="hover:bg-white/80 w-full bg-white/30 flex-[2] min-h-[40px] max-h-24"
        >
          {isPlaying ? (
            <Pause className="h-6 w-6" />
          ) : (
            <Play className="h-6 w-6" />
          )}
        </Button>

        {/* --- Rename Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onOpenRename}
          className="hover:bg-green-300/80 w-full bg-green-300/30 flex-1 min-h-[32px] max-h-12"
        >
          <TextCursorInput className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>

        {/* --- disqualified Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onToggleDisqualified}
          className="hover:bg-red-300/80 w-full bg-red-300/30 flex-1 min-h-[32px] max-h-12"
        >
          <ListX className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>

        {/* --- Rotation Button --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onRotate}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12"
        >
          <RotateCw className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>

        {/* --- close --- */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onClose}
          className="hover:bg-white/80 w-full bg-white/30 flex-1 min-h-[32px] max-h-12"
        >
          <LogOut className="h-4 w-4 sm:h-5 sm:w-5" />
        </Button>
      </div>
    </div>
  );
}
