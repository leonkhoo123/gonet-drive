import { Button } from "@/components/ui/button";
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
      <div className="bg-white/70 rounded-md p-4 w-full max-w-5xl shadow-lg text-black mx-2">
        <h3 className="font-semibold mb-2">Rename File</h3>
        <input
          type="text"
          value={tempName}
          onChange={(e) => {
            onChangeName(e.target.value);
          }}
          className="w-full border-2 border-white/30 p-2 rounded mb-4"
          placeholder="New Video Name"
          autoFocus
        />
        <div className="flex justify-between">
          <Button
            onClick={onDefault}
            className="bg-gray-200/80 hover:bg-gray-300 text-black"
          >
            Default
          </Button>
          <div className="space-x-2">
            <Button
              onClick={onCancel}
              className="bg-gray-300/80 hover:bg-gray-400 text-black"
            >
              Cancel
            </Button>
            <Button
              onClick={onSave}
              className="bg-primary/80 hover:bg-primary text-primary-foreground"
            >
              Save
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
