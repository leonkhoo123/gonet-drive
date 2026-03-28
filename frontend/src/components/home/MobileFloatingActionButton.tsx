import { useRef } from "react";
import { Plus, FolderPlus, Upload, FolderUp } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface MobileFloatingActionButtonProps {
  isHidden: boolean;
  onCreateFolder: () => void;
  onUploadFiles: (files: File[]) => Promise<void> | void;
}

export function MobileFloatingActionButton({
  isHidden,
  onCreateFolder,
  onUploadFiles,
}: MobileFloatingActionButtonProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      void onUploadFiles(Array.from(e.target.files));
    }
    // reset input so the same files can be selected again if needed
    if (fileInputRef.current) fileInputRef.current.value = "";
    if (folderInputRef.current) folderInputRef.current.value = "";
  };

  if (isHidden) return null;

  return (
    <div className="md:hidden absolute bottom-4 right-4 z-50">
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
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <div
            role="button"
            className="w-14 h-14 bg-background text-foreground shadow-lg rounded-full border flex items-center justify-center cursor-pointer hover:bg-muted transition-all duration-300"
          >
            <Plus className="w-7 h-7" />
          </div>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" sideOffset={12}>
          <DropdownMenuItem onClick={onCreateFolder}>
            <FolderPlus className="mr-2 h-4 w-4" />
            <span>Create Folder</span>
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => fileInputRef.current?.click()}>
            <Upload className="mr-2 h-4 w-4" />
            <span>Upload File</span>
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => folderInputRef.current?.click()}>
            <FolderUp className="mr-2 h-4 w-4" />
            <span>Upload Folder</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
