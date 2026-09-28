import { Button } from "@/components/ui/button";
import { Check, Plus, Redo2, Undo2, X } from "lucide-react";
import { CONTROL, CONTROL_ACTIVE, CONTROL_BLUE, CONTROL_GREEN } from "./controlStyles";
import type { EventEditToolbarProps } from "./types";

/**
 * Top bar shown while the event editor is open. Holds the global actions
 * (tag, undo/redo, cancel, save) and the event count. Sits above the video so it
 * stays reachable on touch without colliding with the transport column.
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
    <div className="absolute top-3 left-1/2 -translate-x-1/2 z-40 flex items-center gap-1 rounded-md bg-gray-100/95 px-2 py-1 text-gray-900 shadow-lg">
      <span className="px-1 text-xs font-semibold tabular-nums whitespace-nowrap">
        {String(count)} event{count === 1 ? "" : "s"}
      </span>

      <Button
        variant="ghost"
        size="sm"
        onClick={onAdd}
        disabled={!canAdd}
        title={
          canAdd
            ? "Add an event at the playhead"
            : "Can't add on an event boundary"
        }
        aria-label="Add an event at the playhead"
        className={`${CONTROL_BLUE} h-8 px-2 text-xs disabled:opacity-40`}
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
        className={`${CONTROL} h-8 w-8 disabled:opacity-40`}
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
        className={`${CONTROL} h-8 w-8 disabled:opacity-40`}
      >
        <Redo2 className="h-4 w-4" />
      </Button>

      <Button
        variant="ghost"
        size="sm"
        onClick={onCancel}
        disabled={isSaving}
        title="Discard changes"
        aria-label="Discard changes"
        className={`${CONTROL} h-8 px-2 text-xs`}
      >
        <X className="h-4 w-4 mr-1" /> Cancel
      </Button>

      <Button
        variant="ghost"
        size="sm"
        onClick={onSave}
        disabled={isSaving}
        title="Save events"
        aria-label="Save events"
        className={`${isSaving ? CONTROL_ACTIVE : CONTROL_GREEN} h-8 px-2 text-xs`}
      >
        <Check className="h-4 w-4 mr-1" /> {isSaving ? "Saving…" : "Done"}
      </Button>
    </div>
  );
}
