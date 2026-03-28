import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Download } from "lucide-react";

interface HomeDownloadDirDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  currentPath: string;
}

export default function HomeDownloadDirDialog({
  isOpen,
  onOpenChange,
  onConfirm,
  currentPath,
}: HomeDownloadDirDialogProps) {
  const folderName = currentPath === "/" ? "Root Directory" : currentPath.split("/").filter(Boolean).pop() ?? "Directory";

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5 text-primary" />
            Download Folder
          </DialogTitle>
          <DialogDescription className="py-4">
            Are you sure you want to download <strong>{folderName}</strong> as a zip file? This might take some time for large directories.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex gap-2 sm:justify-end">
          <Button variant="outline" onClick={() => { onOpenChange(false); }}>
            Cancel
          </Button>
          <Button variant="default" onClick={onConfirm}>
            Download
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
