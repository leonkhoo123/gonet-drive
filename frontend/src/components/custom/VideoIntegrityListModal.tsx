import { useState, useEffect, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Film,
} from "lucide-react";
import {
  getList,
  type VideoIntegrityEntry,
} from "@/api/api-video-integrity";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PAGE_SIZE = 100;

export function VideoIntegrityListModal({ open, onOpenChange }: Props) {
  const [entries, setEntries] = useState<VideoIntegrityEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(0);
  const [imgErrors, setImgErrors] = useState<Set<string>>(
    () => new Set<string>()
  );

  const fetchList = useCallback(() => {
    setLoading(true);
    getList()
      .then((data) => {
        setEntries(data.entries);
      })
      .catch(() => {
        // silently fail
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    if (open) {
      fetchList();
      setPage(0);
      setImgErrors(new Set<string>());
    }
  }, [open, fetchList]);

  const totalPages = Math.max(1, Math.ceil(entries.length / PAGE_SIZE));
  const pagedEntries = entries.slice(
    page * PAGE_SIZE,
    (page + 1) * PAGE_SIZE
  );

  const handleImgError = (hash: string) => {
    setImgErrors((prev) => {
      const next = new Set(prev);
      next.add(hash);
      return next;
    });
  };

  const thumbnailUrl = (entry: VideoIntegrityEntry) => {
    return `/api/user/video/thumbnail/file/${entry.relative_path}`;
  };

  const fileName = (entry: VideoIntegrityEntry) => {
    const parts = entry.relative_path.replace(/\\/g, "/").split("/");
    return parts[parts.length - 1] || entry.relative_path;
  };

  const parentPath = (entry: VideoIntegrityEntry) => {
    const parts = entry.relative_path.replace(/\\/g, "/").split("/");
    parts.pop();
    return parts.join("/") + "/";
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl w-[95vw] max-w-2xl mx-auto max-h-[85vh] flex flex-col">
        <DialogHeader className="shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-amber-500" />
            Corrupt Video Files
          </DialogTitle>
          <DialogDescription>
            {entries.length} file{entries.length !== 1 ? "s" : ""} with
            corrupt avcC container detected.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto -mx-6 px-6">
          {loading ? (
            <div className="space-y-3 py-2">
              {Array.from({ length: 5 }, (_, i) => (
                <div
                  key={`vi-skel-${String(i)}`}
                  className="flex items-center gap-3 py-3"
                >
                  <Skeleton className="h-12 w-12 rounded-lg shrink-0" />
                  <div className="space-y-1.5 flex-1">
                    <Skeleton className="h-4 w-48" />
                    <Skeleton className="h-3 w-64" />
                  </div>
                </div>
              ))}
            </div>
          ) : entries.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Film className="h-12 w-12 mb-3 opacity-30" />
              <p className="text-sm">No corrupt video files detected.</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {pagedEntries.map((entry) => (
                <div
                  key={entry.hash}
                  className="flex items-center gap-3 py-3"
                >
                  {/* Thumbnail */}
                  <div className="h-12 w-12 rounded-lg overflow-hidden bg-muted shrink-0 flex items-center justify-center">
                    {!imgErrors.has(entry.hash) ? (
                      <img
                        src={thumbnailUrl(entry)}
                        alt={fileName(entry)}
                        className="h-full w-full object-cover"
                        onError={() => { handleImgError(entry.hash); }}
                        loading="lazy"
                      />
                    ) : (
                      <Film className="h-6 w-6 text-muted-foreground" />
                    )}
                  </div>

                  {/* Name + Path */}
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-foreground truncate">
                      {fileName(entry)}
                    </p>
                    <p className="text-xs text-muted-foreground truncate">
                      {parentPath(entry)}
                    </p>
                    <p className="text-xs text-muted-foreground/70 mt-0.5">
                      {entry.mime_codec_string}
                      {" · "}
                      {new Date(entry.detected_at).toLocaleDateString()}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Pagination */}
        {entries.length > PAGE_SIZE && (
          <div className="flex items-center justify-between pt-3 border-t shrink-0">
            <span className="text-sm text-muted-foreground">
              {page * PAGE_SIZE + 1}–
              {Math.min((page + 1) * PAGE_SIZE, entries.length)}{" "}
              of {entries.length}
            </span>
            <div className="flex items-center gap-1">
              <Button
                variant="outline"
                size="sm"
                onClick={() => { setPage((p) => Math.max(0, p - 1)); }}
                disabled={page === 0}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setPage((p) => Math.min(totalPages - 1, p + 1));
                }}
                disabled={page >= totalPages - 1}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
