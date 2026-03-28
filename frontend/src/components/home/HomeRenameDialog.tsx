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

interface HomeRenameDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (newName: string) => void | Promise<void>;
  itemToRename: string | null;
  isLoading: boolean;
  existingNames: string[];
}

export default function HomeRenameDialog({
  isOpen,
  onOpenChange,
  onConfirm,
  itemToRename,
  isLoading,
  existingNames,
}: HomeRenameDialogProps) {
  const [renameInput, setRenameInput] = useState("");

  useEffect(() => {
    if (isOpen && itemToRename) {
      setRenameInput(itemToRename);
    } else {
      setRenameInput("");
    }
  }, [isOpen, itemToRename]);

  const isRenameExists = existingNames.some(
    (name) => name.toLowerCase() === renameInput.toLowerCase() && name.toLowerCase() !== itemToRename?.toLowerCase()
  );

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Rename Item</DialogTitle>
          <DialogDescription>
            Enter a new name for the item.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <Input 
            value={renameInput}
            onChange={(e) => { setRenameInput(e.target.value); }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !isRenameExists && renameInput && renameInput !== itemToRename) {
                void onConfirm(renameInput);
              }
            }}
            placeholder="New name"
            autoFocus
          />
          {isRenameExists && (
            <p className="text-destructive text-sm mt-2">A file or folder with this name already exists.</p>
          )}
        </div>
        <DialogFooter className="flex gap-2 sm:justify-end">
          <Button variant="outline" onClick={() => { onOpenChange(false); }}>
            Cancel
          </Button>
          <Button 
            onClick={() => { void onConfirm(renameInput); }} 
            disabled={isLoading || !renameInput || renameInput === itemToRename || isRenameExists}
          >
            {isLoading ? "Renaming..." : "Rename"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
