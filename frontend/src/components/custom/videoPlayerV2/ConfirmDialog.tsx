import { Button } from "@/components/ui/button";
import { AlertTriangle } from "lucide-react";
import { CONTROL, CONTROL_GREEN, CONTROL_RED, PANEL } from "./controlStyles";
import type { ConfirmDialogProps } from "./types";

/**
 * Confirmation dialog for the video player. Uses the player's own flat gray
 * surface (`PANEL`) and black overlay so it matches `VideoRenameDialog` and the
 * control column rather than the app's theme-based dialogs.
 *
 * The overlay sits above every editor layer; clicking it cancels, and the host
 * routes Escape/Back through the same cancel path.
 */
export function ConfirmDialog({
  title,
  description,
  confirmLabel,
  destructive,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <div
      className="absolute inset-0 z-50 bg-black/70 flex items-center justify-center p-4"
      onClick={(e) => {
        e.stopPropagation();
        onCancel();
      }}
    >
      <div
        className={`rounded-md p-4 w-full max-w-md mx-2 ${PANEL}`}
        onClick={(e) => {
          e.stopPropagation();
        }}
      >
        <div className="flex items-start gap-3">
          <div className="h-9 w-9 shrink-0 rounded-full bg-gray-300/70 flex items-center justify-center">
            <AlertTriangle className="h-5 w-5 text-amber-600" />
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold">{title}</h3>
            {description && (
              <p className="text-sm text-gray-600 mt-1">{description}</p>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-5">
          <Button
            variant="ghost"
            onClick={onCancel}
            autoFocus
            className={CONTROL}
          >
            Cancel
          </Button>
          <Button
            variant="ghost"
            onClick={onConfirm}
            className={destructive ? CONTROL_RED : CONTROL_GREEN}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
