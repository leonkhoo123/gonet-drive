import { Button } from "@/components/ui/button";
import { Check, Plus, Redo2, RotateCcw, RotateCw, TextCursorInput, Undo2, X } from "lucide-react";
import {
  CONTROL,
  CONTROL_ACTIVE,
  CONTROL_BLUE,
  CONTROL_GREEN,
  CONTROL_GREEN_STRONG,
  CONTROL_RED,
  CONTROL_YELLOW,
} from "./controlStyles";
import type { EventEditToolbarProps } from "./types";

/**
 * Left-hand edit column shown while the event editor is open, mirroring the
 * playback column on the right. Holds the global actions (tag, undo/redo,
 * cancel, save), the event count, and the "Review" box (rotate / rename) styled
 * like the edit timeline so the two columns frame the video without shouting.
 */
export function EventEditToolbar({
  count,
  isSaving,
  canUndo,
  canRedo,
  canAdd,
  eventsDirty,
  rotation,
  isRenamed,
  hasChanges,
  onAdd,
  onUndo,
  onRedo,
  onRevert,
  onRotate,
  onOpenRename,
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
        {/* Event card: tag + history + revert-to-loaded, mirroring the Review
            card. Lights up green while the event draft differs from loaded. */}
        <div
          className={`flex w-full shrink-0 flex-col rounded-md border bg-white/5 p-1 transition-colors ${
            eventsDirty ? "border-emerald-400/70" : "border-white/15"
          }`}
        >
          <span className="mb-1 text-center text-[10px] font-semibold uppercase tracking-wide text-white/50">
            Event
          </span>
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
            className={`${CONTROL_BLUE} h-8 w-full min-w-0 min-h-[32px] px-1 text-[11px] disabled:opacity-40`}
          >
            <Plus className="h-3.5 w-3.5 mr-1" /> Event
          </Button>

          <Button
            variant="ghost"
            onClick={onUndo}
            disabled={!canUndo}
            title="Undo"
            aria-label="Undo"
            className={`${CONTROL} mt-1 h-8 w-full min-w-0 min-h-[32px] disabled:opacity-40`}
          >
            <Undo2 className="h-4 w-4" />
          </Button>

          <Button
            variant="ghost"
            onClick={onRedo}
            disabled={!canRedo}
            title="Redo"
            aria-label="Redo"
            className={`${CONTROL} mt-1 h-8 w-full min-w-0 min-h-[32px] disabled:opacity-40`}
          >
            <Redo2 className="h-4 w-4" />
          </Button>

          <Button
            variant="ghost"
            onClick={onRevert}
            disabled={!eventsDirty}
            title="Revert to the loaded events"
            aria-label="Revert to the loaded events"
            className={`${CONTROL} mt-1 h-8 w-full min-w-0 min-h-[32px] px-1 text-[11px] disabled:opacity-40`}
          >
            <RotateCcw className="h-3.5 w-3.5 mr-1" /> Revert
          </Button>
        </div>

        {/* Review box: rotate/rename fold into the same Done action. Kept below
            the edit actions and above Cancel/Done, so the terminal actions read
            as "review, then cancel or commit". Lights up green while a rename is
            staged (the new name shows at the seek bar). */}
        <div
          className={`flex w-full shrink-0 flex-col rounded-md border bg-white/5 p-1 transition-colors ${
            isRenamed ? "border-emerald-400/70" : "border-white/15"
          }`}
        >
          <span className="mb-1 text-center text-[10px] font-semibold uppercase tracking-wide text-white/50">
            Review
          </span>
          <Button
            variant="ghost"
            onClick={onRotate}
            title="Rotate the video"
            aria-label="Rotate the video"
            className={`${CONTROL_YELLOW} h-8 w-full min-w-0 min-h-[32px] px-1 text-[11px]`}
          >
            <RotateCw className="h-3.5 w-3.5 mr-1" /> Rotate
          </Button>
          {/* Always rendered (shows "0°") so the box never changes height. */}
          <span className="mt-0.5 min-h-[14px] truncate text-center text-[10px] leading-[14px] tabular-nums text-white/60">
            {String(rotation)}°
          </span>
          <Button
            variant="ghost"
            onClick={onOpenRename}
            title="Rename the file (sends it to Done when you save)"
            aria-label="Rename the file"
            aria-pressed={isRenamed}
            className={`${CONTROL_GREEN} h-8 w-full min-w-0 min-h-[32px] px-1 text-[11px]`}
          >
            <TextCursorInput className="h-3.5 w-3.5 mr-1" /> Rename
          </Button>
        </div>

        {/* Terminal actions pinned to the bottom, below the Review box. */}
        <Button
          variant="ghost"
          onClick={onCancel}
          disabled={isSaving}
          title="Discard changes"
          aria-label="Discard changes"
          className={`${CONTROL_RED} w-full flex-1 min-h-[40px] max-h-14 px-1 text-xs`}
        >
          <X className="h-4 w-4 mr-1" /> Cancel
        </Button>

        <Button
          variant="ghost"
          onClick={onSave}
          disabled={isSaving}
          title={hasChanges ? "Save events" : "Leave the editor"}
          aria-label={hasChanges ? "Save events" : "Leave the editor"}
          className={`${
            isSaving
              ? CONTROL_ACTIVE
              : hasChanges
                ? CONTROL_GREEN_STRONG
                : CONTROL_GREEN
          } w-full flex-1 min-h-[40px] max-h-14 px-1 text-xs`}
        >
          <Check className="h-4 w-4 mr-1" /> {isSaving ? "Saving…" : "Done"}
        </Button>
      </div>
    </div>
  );
}
