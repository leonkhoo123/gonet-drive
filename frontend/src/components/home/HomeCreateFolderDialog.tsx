import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface HomeCreateFolderDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (folderName: string) => void | Promise<void>;
  isLoading: boolean;
  existingNames: string[];
}

export default function HomeCreateFolderDialog({
  isOpen,
  onOpenChange,
  onConfirm,
  isLoading,
  existingNames,
}: HomeCreateFolderDialogProps) {
  const [folderInput, setFolderInput] = useState("");

  useEffect(() => {
    if (isOpen) {
      setFolderInput("New Folder");
    } else {
      setFolderInput("");
    }
  }, [isOpen]);

  const isFolderExists = existingNames.some(
    (name) => name.toLowerCase() === folderInput.toLowerCase()
  );

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create Folder</DialogTitle>
          <DialogDescription>
            Enter a name for the new folder.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <Input 
            value={folderInput}
            onChange={(e) => { setFolderInput(e.target.value); }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !isFolderExists && folderInput) {
                void onConfirm(folderInput);
              }
            }}
            placeholder="Folder name"
            autoFocus
          />
          {isFolderExists && (
            <p className="text-destructive text-sm mt-2">A file or folder with this name already exists.</p>
          )}
        </div>
        <DialogFooter className="flex gap-2 sm:justify-end">
          <Button variant="outline" onClick={() => { onOpenChange(false); }}>
            Cancel
          </Button>
          <Button 
            onClick={() => { void onConfirm(folderInput); }} 
            disabled={isLoading || !folderInput || isFolderExists}
          >
            {isLoading ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
