import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { PropertiesResponse } from "@/api/api-file";
import { formatBytes, formatLastModified } from "@/utils/utils";
import { File, Folder, Files, Loader2 } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

interface SharePropertiesModalProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  data: PropertiesResponse | null;
}

export default function SharePropertiesModal({ isOpen, onOpenChange, data }: SharePropertiesModalProps) {
  const renderLoadingState = () => (
    <div className="flex flex-col gap-4 py-4">
      <div className="flex items-center gap-4 w-full">
        <Skeleton className="w-12 h-12 rounded-lg" />
        <Skeleton className="h-6 w-1/2" />
      </div>
      <Separator />
      <div className="grid grid-cols-[80px_1fr] sm:grid-cols-[100px_1fr] gap-4 w-full">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-2/3" />
        <Separator className="col-span-2 my-2" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-2/3" />
      </div>
    </div>
  );

  const renderContent = () => {
    if (!data) return renderLoadingState();

    const isMultiple = data.type === "multiple";
    const isDir = data.type === "directory";

    const getIcon = () => {
      if (isMultiple) return <Files className="w-12 h-12 text-primary" />;
      if (isDir) return <Folder className="w-12 h-12 text-primary fill-primary/20" />;
      return <File className="w-12 h-12 text-gray-500" />;
    };

    return (
      <div className="flex flex-col gap-4 py-4 max-h-[60vh] overflow-y-auto overflow-x-hidden">
        <div className="flex items-center gap-4 w-full">
          <div className="flex-shrink-0">
            {getIcon()}
          </div>
          <span className="text-xl font-medium break-all line-clamp-3">
            {isMultiple ? "Multiple Items" : data.name}
          </span>
        </div>

        <Separator />

        <div className="grid grid-cols-[80px_1fr] sm:grid-cols-[100px_1fr] gap-2 text-sm w-full">
          <span className="font-semibold text-muted-foreground">Type:</span>
          <span>
            {isMultiple ? "Multiple Items" : isDir ? "Folder" : "File"}
          </span>

          <span className="font-semibold text-muted-foreground">Location:</span>
          <span className="break-all">{data.location}</span>

          <span className="font-semibold text-muted-foreground">Size:</span>
          <span>
            {formatBytes(data.totalSizeBytes)} ({data.totalSizeBytes.toLocaleString()} bytes)
          </span>

          {(isDir || isMultiple) && (
            <>
              <span className="font-semibold text-muted-foreground">Contains:</span>
              <span>
                {data.contains.files} Files, {data.contains.folders} Folders
              </span>
            </>
          )}

          <Separator className="col-span-2 my-2" />

          {!isMultiple && (
            <>
              <span className="font-semibold text-muted-foreground">Created:</span>
              <span>{data.createdAt ? formatLastModified(data.createdAt) : "Unknown"}</span>

              <span className="font-semibold text-muted-foreground">Modified:</span>
              <span>{data.modifiedAt ? formatLastModified(data.modifiedAt) : "Unknown"}</span>

              <span className="font-semibold text-muted-foreground">Accessed:</span>
              <span>{data.accessedAt ? formatLastModified(data.accessedAt) : "Unknown"}</span>
            </>
          )}
        </div>
      </div>
    );
  };

  const title = !data
    ? "Info"
    : data.type === "multiple"
    ? "Info (Multiple Items)"
    : data.name
    ? `${data.name} Info`
    : "Info";

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md w-[90vw] max-w-lg mx-auto overflow-hidden">
        <DialogHeader>
          <DialogTitle className="break-all line-clamp-2 pr-6 flex items-center gap-2">
            {title}
            {!data && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
          </DialogTitle>
          <DialogDescription className="sr-only">
            Properties and details of the selected item(s)
          </DialogDescription>
        </DialogHeader>

        {renderContent()}

        <DialogFooter>
          <Button onClick={() => { onOpenChange(false); }}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
