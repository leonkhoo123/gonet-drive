import { Button } from "@/components/ui/button";
import { Crosshair, Trash2 } from "lucide-react";
import { formatTime } from "@/utils/videoPlayerV2Events";
import { CONTROL, CONTROL_RED, PANEL } from "./controlStyles";
import type { EventInspectorProps } from "./types";

/** Format seconds as `m:ss.d` for edge-precision editing. */
const preciseTime = (t: number): string => {
  const tenths = Math.min(9, Math.round((t % 1) * 10));
  return `${formatTime(t)}.${String(tenths)}`;
};

function EdgeRow({
  label,
  value,
  onNudge,
  onSetToPlayhead,
}: {
  label: string;
  value: number;
  onNudge: (delta: number) => void;
  onSetToPlayhead: () => void;
}) {
  return (
    <div className="flex w-full items-center gap-1">
      <span className="w-10 shrink-0 text-xs opacity-70">{label}</span>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => { onNudge(-1); }}
        className={`${CONTROL} h-8 flex-1 px-2 text-xs tabular-nums`}
        aria-label={`${label} minus one second`}
      >
        −1
      </Button>
      <span className="w-16 shrink-0 text-center text-sm font-semibold tabular-nums">
        {preciseTime(value)}
      </span>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => { onNudge(1); }}
        className={`${CONTROL} h-8 flex-1 px-2 text-xs tabular-nums`}
        aria-label={`${label} plus one second`}
      >
        +1
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={onSetToPlayhead}
        title={`Set ${label.toLowerCase()} to playhead`}
        aria-label={`Set ${label.toLowerCase()} to playhead`}
        className={`${CONTROL} h-8 w-8 shrink-0`}
      >
        <Crosshair className="h-4 w-4" />
      </Button>
    </div>
  );
}

/**
 * Selected-event inspector, docked above the edit bar. Large, pixel-free nudge
 * buttons give touch users (and anyone at low zoom) precision that dragging a
 * thin cap cannot.
 */
export function EventInspector({
  span,
  currentTime,
  onNudge,
  onSetToPlayhead,
  onDelete,
}: EventInspectorProps) {
  if (!span) return null;
  const [start, end] = span;

  return (
    <div className={`absolute bottom-44 left-1/2 -translate-x-1/2 z-40 flex flex-col gap-1 rounded-md px-3 py-2 ${PANEL}`}>
      <div className="flex items-center justify-between gap-4 text-xs">
        <span className="opacity-70">
          Selected · {preciseTime(start)} → {preciseTime(end)}
        </span>
        <span className="opacity-50 tabular-nums">
          dur {(end - start).toFixed(1)}s · playhead {preciseTime(currentTime)}
        </span>
      </div>

      <EdgeRow
        label="Start"
        value={start}
        onNudge={(delta) => { onNudge("start", delta); }}
        onSetToPlayhead={() => { onSetToPlayhead("start"); }}
      />
      <EdgeRow
        label="End"
        value={end}
        onNudge={(delta) => { onNudge("end", delta); }}
        onSetToPlayhead={() => { onSetToPlayhead("end"); }}
      />

      <div className="flex justify-end">
        <Button
          variant="ghost"
          size="sm"
          onClick={onDelete}
          title="Delete event"
          aria-label="Delete event"
          className={`${CONTROL_RED} h-8 px-3 text-xs`}
        >
          <Trash2 className="h-4 w-4 mr-1" /> Delete
        </Button>
      </div>
    </div>
  );
}
