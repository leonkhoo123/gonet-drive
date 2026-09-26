import { Button } from "@/components/ui/button";
import { GLASS, GLASS_GREEN, GLASS_PANEL } from "./glassStyles";
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
      <div className={`rounded-md p-4 w-full max-w-5xl mx-2 ${GLASS_PANEL}`}>
        <h3 className="font-semibold mb-2">Rename File</h3>
        <input
          type="text"
          value={tempName}
          onChange={(e) => {
            onChangeName(e.target.value);
          }}
          className="w-full bg-black/5 p-2 rounded mb-4 text-black placeholder:text-black/50 outline-none focus:ring-2 focus:ring-black/30"
          placeholder="New Video Name"
          autoFocus
        />
        <div className="flex justify-between">
          <Button variant="ghost" onClick={onDefault} className={GLASS}>
            Default
          </Button>
          <div className="space-x-2">
            <Button variant="ghost" onClick={onCancel} className={GLASS}>
              Cancel
            </Button>
            <Button variant="ghost" onClick={onSave} className={GLASS_GREEN}>
              Save
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
