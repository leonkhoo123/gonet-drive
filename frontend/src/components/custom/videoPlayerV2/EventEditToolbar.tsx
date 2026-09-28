import { Button } from "@/components/ui/button";
import { Check, Plus, Redo2, Undo2, X } from "lucide-react";
import { CONTROL, CONTROL_ACTIVE, CONTROL_BLUE, CONTROL_GREEN } from "./controlStyles";
import type { EventEditToolbarProps } from "./types";

/**
 * Left-hand edit column shown while the event editor is open, mirroring the
 * playback column on the right. Holds the global actions (tag, undo/redo,
 * cancel, save) and the event count, styled like the edit timeline so the two
 * columns frame the video without shouting over it.
 */
export function EventEditToolbar({
  count,
  isSaving,
  canUndo,
  canRedo,
  canAdd,
  onAdd,
  onUndo,
  onRedo,
  onCancel,
  onSave,
}: EventEditToolbarProps) {
  return (
    <div className="absolute left-0 top-11 bottom-44 z-40 flex flex-col p-1 lg:p-2 select-none">
      {/* Event count, pinned to the top of the column */}
      <div className="flex shrink-0 justify-center">
        <span className="w-20 lg:w-24 whitespace-nowrap rounded-md bg-white/5 px-1 py-1.5 text-center text-xs font-semibold tabular-nums text-white/90">
          {String(count)} event{count === 1 ? "" : "s"}
        </span>
      </div>

      <div className="flex-1 min-h-0 flex flex-col gap-1 lg:gap-2 overflow-y-auto scrollbar-hide w-20 lg:w-24 py-2 justify-center">
        <Button
          variant="ghost"
          onClick={onAdd}
          disabled={!canAdd}
          title={
            canAdd
              ? "Add an event at the playhead"
              : "Can't add on an event boundary"
          }
          aria-label="Add an event at the playhead"
          className={`${CONTROL_BLUE} w-full flex-1 min-h-[40px] max-h-14 px-1 text-xs disabled:opacity-40`}
        >
          <Plus className="h-4 w-4 mr-1" /> Event
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={onUndo}
          disabled={!canUndo}
          title="Undo"
          aria-label="Undo"
          className={`${CONTROL} w-full flex-1 min-h-[36px] max-h-12 disabled:opacity-40`}
        >
          <Undo2 className="h-4 w-4" />
        </Button>

        <Button
          variant="ghost"
          size="icon"
          onClick={onRedo}
          disabled={!canRedo}
          title="Redo"
          aria-label="Redo"
          className={`${CONTROL} w-full flex-1 min-h-[36px] max-h-12 disabled:opacity-40`}
        >
          <Redo2 className="h-4 w-4" />
        </Button>

        <Button
          variant="ghost"
          onClick={onCancel}
          disabled={isSaving}
          title="Discard changes"
          aria-label="Discard changes"
          className={`${CONTROL} w-full flex-1 min-h-[36px] max-h-12 px-1 text-xs`}
        >
          <X className="h-4 w-4 mr-1" /> Cancel
        </Button>

        <Button
          variant="ghost"
          onClick={onSave}
          disabled={isSaving}
          title="Save events"
          aria-label="Save events"
          className={`${isSaving ? CONTROL_ACTIVE : CONTROL_GREEN} w-full flex-1 min-h-[40px] max-h-14 px-1 text-xs`}
        >
          <Check className="h-4 w-4 mr-1" /> {isSaving ? "Saving…" : "Done"}
        </Button>
      </div>
    </div>
  );
}
