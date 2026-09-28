import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { ConfirmState } from "./types";

export function ConfirmOverlay({ state, onCancel }: { state: ConfirmState; onCancel: () => void }) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onCancel();
      }
    };
    window.addEventListener('keydown', handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener('keydown', handleKeyDown, { capture: true });
    };
  }, [onCancel]);

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-150"
      onClick={(e) => { e.stopPropagation(); onCancel(); }}
    >
      <div
        className="w-full max-w-md bg-background border rounded-xl shadow-2xl p-5"
        onClick={(e) => { e.stopPropagation(); }}
      >
        <div className="flex items-start gap-3">
          <div className="h-9 w-9 shrink-0 rounded-full bg-muted flex items-center justify-center">
            <AlertTriangle className="h-5 w-5 text-amber-500" />
          </div>
          <div className="min-w-0">
            <h3 className="text-base font-semibold">{state.title}</h3>
            <p className="text-sm text-muted-foreground mt-1">{state.description}</p>
          </div>
        </div>
        <div className="flex justify-end gap-2 mt-5">
          <Button variant="outline" size="sm" onClick={onCancel}>Cancel</Button>
          <Button
            variant={state.destructive ? "destructive" : "default"}
            size="sm"
            onClick={state.onConfirm}
          >
            {state.confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
