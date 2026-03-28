import { ArrowLeft, CheckSquare, Scissors, Copy, Clipboard, Pencil, Trash2, Plus, RefreshCcw, FolderPlus, Upload, FolderUp, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useRef } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface ShareToolbarProps {
  currentPath: string;
  shareRoot?: string;
  selectedItemsSize: number;
  clipboardItemsCount: number;
  clipboardOperation: 'cut' | 'copy' | null;
  clipboardSourceDir?: string;
  isFolderEmpty?: boolean;
  isSingleFile?: boolean;
  onBack: () => void;
  onSelectAll: () => void;
  onCut: () => void;
  onCopy: () => void;
  onPaste: () => void;
  onRename: () => void;
  onDelete: () => void;
  onDownload: () => void;
  onRefresh: () => void;
  onCreateFolder: () => void;
  onUploadFiles?: (files: File[]) => void;
  isRecycleBinSelected?: boolean;
  authority: string;
}

export default function ShareToolbar({
  currentPath,
  shareRoot,
  selectedItemsSize,
  clipboardItemsCount,
  clipboardOperation,
  clipboardSourceDir,
  isFolderEmpty = false,
  isSingleFile = false,
  onBack,
  onSelectAll,
  onCut,
  onCopy,
  onPaste,
  onRename,
  onDelete,
  onDownload,
  onRefresh,
  onCreateFolder,
  onUploadFiles,
  isRecycleBinSelected = false,
  authority,
}: ShareToolbarProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  const isRecycleBin = currentPath === '/.cloud_delete' || currentPath.startsWith('/.cloud_delete/');

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0 && onUploadFiles) {
      onUploadFiles(Array.from(e.target.files));
    }
    // reset input so the same files can be selected again if needed
    if (fileInputRef.current) fileInputRef.current.value = "";
    if (folderInputRef.current) folderInputRef.current.value = "";
  };

  return (
    <div className="hidden md:flex items-center gap-1 p-2 px-4 border-b bg-muted/5 shrink-0 overflow-x-auto">
      <input 
        type="file" 
        multiple 
        className="hidden" 
        ref={fileInputRef} 
        onChange={handleFileChange} 
      />
      <input 
        type="file" 
        className="hidden" 
        ref={folderInputRef} 
        {...{ webkitdirectory: "", directory: "" } as any} 
        onChange={handleFileChange} 
      />

      <Button variant="ghost" size="sm" onClick={onBack} disabled={currentPath === "/" || currentPath === shareRoot} className="h-8 px-2" title="Back">
        <ArrowLeft className="h-4 w-4 md:mr-1" />
        <span className="hidden md:inline">Back</span>
      </Button>

      <div className="h-4 w-[1px] bg-border mx-1" />

      <Button variant="ghost" size="sm" onClick={onSelectAll} className="h-8 w-8 p-0" title="Select All">
        <CheckSquare className="h-4 w-4" />
      </Button>
      {authority === 'modify' && (
        <>
          <Button variant="ghost" size="sm" onClick={onCut} disabled={selectedItemsSize === 0 || isRecycleBinSelected} className="h-8 w-8 p-0" title="Cut">
            <Scissors className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={onCopy} disabled={selectedItemsSize === 0 || isRecycleBin || isRecycleBinSelected} className="h-8 w-8 p-0" title="Copy">
            <Copy className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={onPaste} disabled={clipboardItemsCount === 0 || (clipboardOperation === 'cut' && clipboardSourceDir === currentPath) || isRecycleBin || isSingleFile} className="h-8 w-8 p-0" title="Paste">
            <Clipboard className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={onRename} disabled={selectedItemsSize !== 1 || isRecycleBin || isRecycleBinSelected} className="h-8 w-8 p-0" title="Rename">
            <Pencil className="h-4 w-4" />
          </Button>
        </>
      )}
      {authority === 'modify' && (
        <Button variant="ghost" size="sm" onClick={onDelete} disabled={selectedItemsSize === 0 || isRecycleBinSelected} className="h-8 w-8 p-0 text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30" title="Delete">
          <Trash2 className="h-4 w-4" />
        </Button>
      )}
      <Button variant="ghost" size="sm" onClick={onDownload} disabled={(selectedItemsSize === 0 && isFolderEmpty) || isRecycleBinSelected} className="h-8 w-8 p-0" title="Download">
        <Download className="h-4 w-4" />
      </Button>

      {authority === 'modify' && (
        <>
          <div className="h-4 w-[1px] bg-border mx-1" />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 px-2" title="New" disabled={isSingleFile}>
                <Plus className="h-4 w-4 md:mr-1" />
                <span className="hidden md:inline">New</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem onClick={onCreateFolder} disabled={isRecycleBin}>
                <FolderPlus className="mr-2 h-4 w-4" />
                <span>Create Folder</span>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => fileInputRef.current?.click()} disabled={isRecycleBin}>
                <Upload className="mr-2 h-4 w-4" />
                <span>Upload File</span>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => folderInputRef.current?.click()} disabled={isRecycleBin}>
                <FolderUp className="mr-2 h-4 w-4" />
                <span>Upload Folder</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </>
      )}

      <div className="flex-1" />

      <Button variant="ghost" size="sm" onClick={onRefresh} className="h-8 w-8 p-0 text-muted-foreground" title="Refresh">
        <RefreshCcw className="h-4 w-4" />
      </Button>
    </div>
  );
}
