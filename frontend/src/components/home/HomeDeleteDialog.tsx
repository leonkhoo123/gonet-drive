import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface HomeDeleteDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void | Promise<void>;
  itemsToDeleteSize: number;
  currentPath: string;
  isLoading: boolean;
}

export default function HomeDeleteDialog({
  isOpen,
  onOpenChange,
  onConfirm,
  itemsToDeleteSize,
  currentPath,
  isLoading,
}: HomeDeleteDialogProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete Items</DialogTitle>
          <DialogDescription>
            {currentPath.startsWith("/.cloud_delete") 
              ? `Are you sure you want to permanently delete ${itemsToDeleteSize} item(s)? This action cannot be undone.`
              : `Are you sure you want to move ${itemsToDeleteSize} item(s) to the recycle bin?`}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="mt-4 flex gap-2 sm:justify-end">
          <Button variant="outline" onClick={() => { onOpenChange(false); }}>
            Cancel
          </Button>
          <Button autoFocus variant="destructive" onClick={() => { void onConfirm(); }} disabled={isLoading}>
            {isLoading ? "Deleting..." : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
