import { useState, useCallback, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ShieldCheck,
  Play,
  Square,
  List,
  Info,
  Clock,
  AlertTriangle,
} from "lucide-react";
import {
  startScan,
  stopScan,
  getStatus,
  type VideoIntegrityStatus,
} from "@/api/api-video-integrity";
import { toast } from "sonner";
import { VideoIntegrityListModal } from "./VideoIntegrityListModal";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function VideoIntegrityModal({ open, onOpenChange }: Props) {
  const [status, setStatus] = useState<VideoIntegrityStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [listOpen, setListOpen] = useState(false);

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    try {
      const s = await getStatus();
      setStatus(s);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      void fetchStatus();
    }
  }, [open, fetchStatus]);

  const handleStartScan = () => {
    setStarting(true);
    startScan()
      .then(() => {
        toast.success("Video integrity scan started");
        onOpenChange(false);
      })
      .catch((err: unknown) => {
        const msg =
          err instanceof Error ? err.message : "Failed to start scan";
        toast.error(msg);
      })
      .finally(() => {
        setStarting(false);
      });
  };

  const handleStopScan = () => {
    setStopping(true);
    stopScan()
      .then(() => {
        toast.success("Scan stopped");
      })
      .catch((err: unknown) => {
        const msg =
          err instanceof Error ? err.message : "Failed to stop scan";
        toast.error(msg);
      })
      .finally(() => {
        setStopping(false);
        void fetchStatus();
      });
  };

  const lastScanText = status?.last_scan
    ? new Date(status.last_scan).toLocaleString()
    : "Never";

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-lg w-[92vw] max-w-xl mx-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-lg">
              <ShieldCheck className="h-5 w-5 text-primary" />
              Video Integrity Check
            </DialogTitle>
            <DialogDescription>
              Detect corrupt video containers that may cause playback failures
              on iOS and iPad devices.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Status Card */}
            <div className="rounded-lg border bg-card p-4">
              <h4 className="text-sm font-medium text-muted-foreground mb-3">
                Current Status
              </h4>
              {loading ? (
                <div className="space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-4 w-24" />
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-sm">
                    <AlertTriangle className="h-4 w-4 text-amber-500" />
                    <span>
                      Corrupt files detected:{" "}
                      <span className="font-semibold">
                        {status?.corrupt_count ?? 0}
                      </span>
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Clock className="h-4 w-4" />
                    {status?.scan_running ? (
                      <span className="text-primary font-medium">
                        Scan in progress&hellip;
                      </span>
                    ) : (
                      <span>Last scan: {lastScanText}</span>
                    )}
                  </div>
                  {status?.scan_running && (
                    <div className="flex items-center gap-2 text-sm text-primary">
                      <span className="inline-block h-2 w-2 rounded-full bg-primary animate-pulse" />
                      Check the progress panel for details
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Explanation */}
            <div className="rounded-lg border bg-muted/30 p-4 text-sm space-y-3">
              <div className="flex items-start gap-2">
                <Info className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                <div className="space-y-2 text-muted-foreground">
                  <p>
                    <strong>Why scan?</strong> Some MP4/MOV files have
                    a corrupt{" "}
                    <code className="text-xs bg-muted px-1 rounded">avcC</code>{" "}
                    (AVC Decoder Configuration Record) box. This is the most
                    common cause of video playback failure on iPads and iPhones.
                  </p>
                  <p>
                    <strong>When to scan?</strong> After uploading new videos,
                    or whenever you notice that a video won&apos;t play on an
                    iOS device.
                  </p>
                  <p>
                    <strong>How to fix?</strong> Corrupt videos can be repaired
                    by remuxing with ffmpeg. The scan only detects issues — it
                    does not modify your files. Use a tool like ffmpeg with the{" "}
                    <code className="text-xs bg-muted px-1 rounded">
                      h264_metadata
                    </code>{" "}
                    bitstream filter to rebuild the avcC box without
                    re-encoding.
                  </p>
                </div>
              </div>
            </div>
          </div>

          <Separator />

          <DialogFooter className="flex-col sm:flex-row gap-2">
            {(status?.corrupt_count ?? 0) > 0 && (
              <Button
                variant="outline"
                onClick={() => { setListOpen(true); }}
                className="sm:mr-auto"
              >
                <List className="mr-2 h-4 w-4" />
                View Issues ({status?.corrupt_count})
              </Button>
            )}
            <Button
              variant="outline"
              onClick={() => { onOpenChange(false); }}
            >
              Close
            </Button>
            <Button
              onClick={status?.scan_running ? handleStopScan : handleStartScan}
              disabled={starting || stopping}
              variant={status?.scan_running ? "destructive" : "default"}
            >
              {starting || stopping ? (
                <>
                  <span className="inline-block h-4 w-4 rounded-full border-2 border-current border-t-transparent animate-spin mr-2" />
                  {starting ? "Starting..." : "Stopping..."}
                </>
              ) : status?.scan_running ? (
                <>
                  <Square className="mr-2 h-4 w-4" />
                  Stop Scanning
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4" />
                  Start Full Scan
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <VideoIntegrityListModal
        open={listOpen}
        onOpenChange={setListOpen}
      />
    </>
  );
}
