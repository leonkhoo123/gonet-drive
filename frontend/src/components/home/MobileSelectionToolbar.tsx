import { ArrowLeft, CheckSquare, Scissors, Copy, Trash2, Info, Download } from "lucide-react";
import { Button } from "@/components/ui/button";

interface MobileSelectionToolbarProps {
  selectedItemsSize: number;
  onCancel: () => void;
  onSelectAll: () => void;
  onCut: () => void;
  onCopy: () => void;
  onDelete: () => void;
  onProperties: () => void;
  onDownload: () => void;
  isRecycleBin?: boolean;
  isRecycleBinSelected?: boolean;
}

export default function MobileSelectionToolbar({
  selectedItemsSize,
  onCancel,
  onSelectAll,
  onCut,
  onCopy,
  onDelete,
  onProperties,
  onDownload,
  isRecycleBin = false,
  isRecycleBinSelected = false,
}: MobileSelectionToolbarProps) {
  if (selectedItemsSize === 0) return null;

  return (
    <div className="md:hidden h-16 flex items-center justify-between px-2 border-b bg-background shrink-0 overflow-x-auto">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={onCancel} className="h-12 px-2" title="Cancel Selection">
          <ArrowLeft className="h-8 w-8 mr-2" />
          <span className="font-semibold text-lg">{selectedItemsSize} item(s)</span>
        </Button>
      </div>

      <div className="flex items-center gap-0">
        <Button variant="ghost" size="sm" onClick={onSelectAll} className="h-12 w-10 p-0" title="Select All">
          <CheckSquare className="h-8 w-8" />
        </Button>
        <Button variant="ghost" size="sm" onClick={onCut} disabled={isRecycleBinSelected} className="h-12 w-10 p-0" title="Cut">
          <Scissors className="h-8 w-8" />
        </Button>
        <Button variant="ghost" size="sm" onClick={onCopy} disabled={isRecycleBin || isRecycleBinSelected} className="h-12 w-10 p-0" title="Copy">
          <Copy className="h-8 w-8" />
        </Button>
        <Button variant="ghost" size="sm" onClick={onProperties} className="h-12 w-10 p-0" title="Info">
          <Info className="h-8 w-8" />
        </Button>
        <Button variant="ghost" size="sm" onClick={onDownload} disabled={isRecycleBinSelected} className="h-12 w-10 p-0" title="Download">
          <Download className="h-8 w-8" />
        </Button>
        <Button variant="ghost" size="sm" onClick={onDelete} disabled={isRecycleBinSelected} className="h-12 w-10 p-0 text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30" title="Delete">
          <Trash2 className="h-8 w-8" />
        </Button>
      </div>
    </div>
  );
}
