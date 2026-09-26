import { Button } from "@/components/ui/button";
import { CONTROL, CONTROL_GREEN, PANEL } from "./controlStyles";
import type { VideoRenameDialogProps } from "./types";

/** Overlay dialog for renaming the current video. */
export function VideoRenameDialog({
  tempName,
  onChangeName,
  onDefault,
  onCancel,
  onSave,
}: VideoRenameDialogProps) {
  return (
    <div className="absolute inset-0 z-40 bg-black/70 flex items-center justify-center">
      <div className={`rounded-md p-4 w-full max-w-5xl mx-2 ${PANEL}`}>
        <h3 className="font-semibold mb-2">Rename File</h3>
        <input
          type="text"
          value={tempName}
          onChange={(e) => {
            onChangeName(e.target.value);
          }}
          className="w-full rounded bg-white/80 p-2 mb-4 text-base text-gray-900 placeholder:text-gray-500 outline-none"
          placeholder="New Video Name"
          autoFocus
        />
        <div className="flex justify-between">
          <Button variant="ghost" onClick={onDefault} className={CONTROL}>
            Default
          </Button>
          <div className="space-x-2">
            <Button variant="ghost" onClick={onCancel} className={CONTROL}>
              Cancel
            </Button>
            <Button variant="ghost" onClick={onSave} className={CONTROL_GREEN}>
              Save
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
