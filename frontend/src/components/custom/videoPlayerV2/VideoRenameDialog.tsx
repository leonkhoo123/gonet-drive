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
          className="w-full rounded-md border border-white/15 bg-black/50 p-2 mb-4 text-base text-white placeholder:text-white/40 outline-none focus:border-white/40 focus:ring-2 focus:ring-white/20"
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
