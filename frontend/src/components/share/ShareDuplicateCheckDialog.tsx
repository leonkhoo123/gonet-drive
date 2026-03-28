import { useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { formatBytes, formatLastModified } from "@/utils/utils";
import type { DuplicateItem, FileDetail } from "@/api/api-file";
import { File, Folder, ArrowRight, ArrowDown, Loader2 } from "lucide-react";
import { TruncatedText } from "@/components/custom/truncatedText";

interface ShareDuplicateCheckDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void | Promise<void>;
  onCancel: () => void;
  duplicates: DuplicateItem[];
  isLoading: boolean;
  isChecking?: boolean;
}

const FileDetailView = ({ label, detail }: { label: string, detail: FileDetail }) => (
  <div className="flex-1 min-w-0 bg-background border rounded-sm p-2 relative">
    <div className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-1">{label}</div>
    <div className="flex items-center space-x-3 pr-2">
      {detail.isDir ? (
        <Folder className="h-6 w-6 shrink-0 text-primary fill-primary/20" />
      ) : (
        <File className="h-6 w-6 shrink-0 text-gray-400" />
      )}
      <div className="flex flex-col min-w-0 w-full text-left">
        <TruncatedText className="text-foreground text-sm font-medium leading-tight mb-0.5" text={detail.name} />
        <TruncatedText 
          className="text-xs text-muted-foreground" 
          text={`${detail.isDir ? "--" : formatBytes(detail.size)} • ${formatLastModified(detail.modifiedAt)}`} 
        />
      </div>
    </div>
  </div>
);

export default function ShareDuplicateCheckDialog({
  isOpen,
  onOpenChange,
  onConfirm,
  onCancel,
  duplicates,
  isLoading,
  isChecking = false,
}: ShareDuplicateCheckDialogProps) {
  useEffect(() => {
    if (isOpen && !isChecking) {
      // Small timeout to ensure DOM is updated
      const timer = setTimeout(() => {
        document.getElementById('duplicate-continue-btn')?.focus();
      }, 50);
      return () => { clearTimeout(timer); };
    }
  }, [isOpen, isChecking]);

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent 
        className={isChecking ? "sm:max-w-md flex flex-col" : "sm:max-w-3xl md:max-w-4xl lg:max-w-5xl max-h-[90vh] flex flex-col w-[95vw]"}
        onOpenAutoFocus={(e) => {
          e.preventDefault();
          document.getElementById('duplicate-continue-btn')?.focus();
        }}
      >
        {isChecking ? (
          <div className="flex flex-col items-center justify-center p-8 space-y-4">
            <DialogTitle className="sr-only">Checking duplicates</DialogTitle>
            <DialogDescription className="sr-only">Please wait while checking for file duplicates</DialogDescription>
            <Loader2 className="h-10 w-10 animate-spin text-primary" />
            <h3 className="text-lg font-medium text-foreground">Checking for duplicates...</h3>
          </div>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>File Name Conflict</DialogTitle>
              <DialogDescription>
                The following items already exist in the destination. Do you want to merge folders and auto rename the duplicate files?
              </DialogDescription>
            </DialogHeader>

            {/* Buttons on top as requested */}
            <div className="flex gap-2 justify-end mb-2 shrink-0">
              <Button variant="outline" onClick={() => { onCancel(); onOpenChange(false); }} disabled={isLoading}>
                Cancel
              </Button>
              <Button id="duplicate-continue-btn" autoFocus variant="default" onClick={() => { void onConfirm(); }} disabled={isLoading}>
                {isLoading ? "Processing..." : "Continue"}
              </Button>
            </div>

            {/* Scrollable list of duplicates */}
            <div className="flex-1 min-h-0 overflow-y-auto overflow-x-hidden border rounded-md p-2 space-y-3 scrollbar-thumb-rounded-full scrollbar-track-rounded-full scrollbar scrollbar-thumb-black/20 dark:scrollbar-thumb-white/20 scrollbar-track-transparent">
              {duplicates.map((filePair, index) => (
                <div key={index} className="flex flex-col md:flex-row items-stretch p-2 rounded-md bg-secondary/80 dark:bg-secondary/40 border border-border gap-2 w-full">
                  <div className="w-full md:flex-1 min-w-0 flex flex-col justify-center">
                    <FileDetailView label="Source (New)" detail={filePair.source} />
                  </div>
                  
                  {/* Desktop arrow */}
                  <div className="hidden md:flex shrink-0 items-center justify-center text-muted-foreground">
                    <ArrowRight className="h-5 w-5" />
                  </div>
                  
                  {/* Mobile arrow */}
                  <div className="md:hidden flex shrink-0 items-center justify-center text-muted-foreground my-[-4px] z-10">
                    <ArrowDown className="h-4 w-4 bg-secondary/80 dark:bg-secondary/40 rounded-full" />
                  </div>

                  <div className="w-full md:flex-1 min-w-0 flex flex-col justify-center">
                    <FileDetailView label="Destination (Existing)" detail={filePair.target} />
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
