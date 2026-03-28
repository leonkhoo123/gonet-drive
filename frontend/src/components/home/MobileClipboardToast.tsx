import { Clipboard, X } from "lucide-react";
import { Button } from "@/components/ui/button";

interface MobileClipboardToastProps {
  clipboardItemsCount: number;
  operation: "copy" | "cut" | null;
  sourceDir: string | null | undefined;
  currentPath: string;
  onPaste: () => Promise<void> | void;
  onClearClipboard: () => void;
}

export function MobileClipboardToast({
  clipboardItemsCount,
  operation,
  sourceDir,
  currentPath,
  onPaste,
  onClearClipboard,
}: MobileClipboardToastProps) {
  if (clipboardItemsCount === 0) return null;

  return (
    <div className="md:hidden absolute bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 bg-popover text-popover-foreground border border-border/50 p-2 pl-4 rounded-full shadow-lg whitespace-nowrap">
      <div className="flex items-center gap-2">
        <Clipboard className="w-5 h-5 text-muted-foreground" />
        <span className="text-base font-medium">
          {clipboardItemsCount} item(s) {operation === 'cut' ? 'cut' : 'copied'}
        </span>
      </div>
      <div className="h-5 w-[1px] bg-border mx-1" />
      <Button
        size="sm"
        variant="default"
        className="h-9 px-6 text-sm rounded-full font-medium"
        onClick={(e) => {
          e.stopPropagation();
          void onPaste();
        }}
        disabled={operation === 'cut' && sourceDir === currentPath}
      >
        Paste
      </Button>
      <button
        className="p-2 rounded-full hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
        onClick={(e) => {
          e.stopPropagation();
          onClearClipboard();
        }}
      >
        <X className="w-5 h-5" />
      </button>
    </div>
  );
}
