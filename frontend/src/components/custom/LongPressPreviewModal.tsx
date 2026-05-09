import React, { useState, useRef, useEffect, useCallback } from "react";
import { X, Volume2, VolumeX, Loader2 } from "lucide-react";
import type { FileInterface } from "@/api/api-file";

interface LongPressPreviewModalProps {
  file: FileInterface | null;
  onClose: () => void;
}

const thumbUrl = (f: FileInterface) =>
  f.media_type === "photo"
    ? f.url.replace("/photo/play/", "/photo/thumbnail/")
    : f.url.replace("/video/play/", "/video/thumbnail/");


const SWIPE_CLOSE_THRESHOLD = 100; // px vertical to trigger close
const SWIPE_CLOSE_VELOCITY = 0.5;  // px/ms flick threshold
const CLOSE_ANIM_DURATION = 200;   // ms

export const LongPressPreviewModal: React.FC<LongPressPreviewModalProps> = ({
  file,
  onClose,
}) => {
  const [muted, setMuted] = useState(true);
  const [imgError, setImgError] = useState(false);
  const [vidError, setVidError] = useState(false);
  const [isClosing, setIsClosing] = useState(false);
  // Two-layer loading: thumbnail first, then original
  const [originalReady, setOriginalReady] = useState(false);
  const [thumbError, setThumbError] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);
  const closeAnimRef = useRef(false);
  const touchStartY = useRef(0);
  const touchStartTime = useRef(0);

  // Reset errors when file changes
  useEffect(() => {
    setImgError(false);
    setVidError(false);
    setMuted(true);
    setIsClosing(false);
    setOriginalReady(false);
    setThumbError(false);
    closeAnimRef.current = false;
  }, [file]);

  // Preload original image via JS Image() – works for both fresh and cached images
  useEffect(() => {
    if (file?.media_type !== "photo") return;
    let cancelled = false;
    const img = new Image();
    const onDone = () => { if (!cancelled) setOriginalReady(true); };
    const onFail = () => { if (!cancelled) setImgError(true); };
    img.onload = onDone;
    img.onerror = onFail;
    img.src = file.url;
    // Cached images may have already loaded before onload was attached
    if (img.complete) {
      if (img.naturalWidth > 0) onDone();
      else onFail();
    }
    return () => {
      cancelled = true;
      img.onload = null;
      img.onerror = null;
    };
  }, [file]);

  // Sync muted state with video element
  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.muted = muted;
    }
  }, [muted]);

  // Autoplay video once data is loaded (muted to satisfy mobile browser policy)
  const handleVideoLoaded = useCallback(() => {
    const vid = videoRef.current;
    if (!vid) return;
    vid.muted = true;
    vid.play().catch(() => { /* muted autoplay should succeed */ });
  }, []);

  // ---- Swipe-down close ----
  const triggerSwipeClose = useCallback(() => {
    if (closeAnimRef.current) return;
    closeAnimRef.current = true;
    setIsClosing(true);
    setTimeout(() => { onClose(); }, CLOSE_ANIM_DURATION);
  }, [onClose]);

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartY.current = e.touches[0].clientY;
    touchStartTime.current = Date.now();
    e.stopPropagation();
  }, []);

  const handleTouchEnd = useCallback((e: React.TouchEvent) => {
    const deltaY = e.changedTouches[0].clientY - touchStartY.current;
    const elapsed = Date.now() - touchStartTime.current;

    // Swipe-down close: distance threshold
    if (deltaY > SWIPE_CLOSE_THRESHOLD) {
      triggerSwipeClose();
      return;
    }
    // Swipe-down close: velocity threshold (flick)
    if (deltaY > 0 && elapsed > 0 && deltaY / elapsed > SWIPE_CLOSE_VELOCITY) {
      triggerSwipeClose();
      return;
    }
    // Tap on backdrop background (not media) propagates to media container onClick → close.
  }, [triggerSwipeClose]);

  const handleCloseClick = useCallback(
    (e: React.MouseEvent | React.TouchEvent) => {
      e.stopPropagation();
      e.preventDefault();
      triggerSwipeClose();
    },
    [triggerSwipeClose]
  );

  const toggleMute = useCallback(
    (e: React.MouseEvent | React.TouchEvent) => {
      e.stopPropagation();
      e.preventDefault();
      setMuted((prev) => !prev);
    },
    []
  );

  if (!file) return null;

  const isVideo = file.media_type === "video";
  const isMedia = file.media_type === "photo" || file.media_type === "video";
  const showVideo = isVideo && !vidError;
  const showImage = isMedia && !imgError && !showVideo;

  return (
    <div
      className={`fixed inset-0 z-[90] bg-black/90 flex flex-col items-center justify-center select-none animate-fade-in transition-opacity duration-200 ease-out ${
        isClosing ? "opacity-0" : "opacity-100"
      }`}
      style={{ touchAction: "none" }}
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      onTouchMove={(e) => { e.stopPropagation(); }}
    >
      {/* Close button - top right */}
      <button
        onClick={handleCloseClick}
        onTouchEnd={handleCloseClick}
        className="absolute top-4 right-4 z-20 p-2 rounded-full bg-black/50 text-white/80 hover:bg-black/70 hover:text-white transition-colors"
        title="Close preview"
      >
        <X className="w-6 h-6" />
      </button>

      {/* Media container — tap anywhere closes preview */}
      <div
        className="flex-1 w-full flex items-center justify-center"
        onClick={(e) => { e.stopPropagation(); triggerSwipeClose(); }}
      >
        {showVideo ? (
          <div className="relative w-[95vw] h-[95vh]">
            <video
              ref={videoRef}
              src={file.url}
              autoPlay
              muted
              loop
              playsInline
              controls={false}
              className="w-full h-full object-contain rounded-lg"
              onError={() => { setVidError(true); }}
              onLoadedData={handleVideoLoaded}
            />
            {/* Mute toggle — bottom-right of the video */}
            <button
              onClick={toggleMute}
              onTouchEnd={toggleMute}
              className="absolute bottom-2 right-2 z-20 p-2 rounded-full bg-black/50 text-white/80 hover:bg-black/70 hover:text-white transition-colors"
              title={muted ? "Unmute" : "Mute"}
            >
              {muted ? (
                <VolumeX className="w-6 h-6" />
              ) : (
                <Volume2 className="w-6 h-6" />
              )}
            </button>
          </div>
        ) : showImage ? (
          <div className="relative w-[95vw] h-[95vh]">
            {/* Visible layer: starts as thumbnail, swaps to original when cached */}
            <img
              src={originalReady ? file.url : thumbUrl(file)}
              alt={file.name}
              className="w-full h-full object-contain rounded-lg"
              draggable={false}
              onError={() => {
                if (!originalReady) setThumbError(true);
                else setImgError(true);
              }}
            />
            {/* Thumbnail error + original not ready */}
            {thumbError && !originalReady && (
              <div className="absolute inset-0 flex items-center justify-center">
                <Loader2 className="w-12 h-12 text-white/70 animate-spin" />
              </div>
            )}
          </div>
        ) : (
          <span className="text-white/50 text-sm">Failed to load preview</span>
        )}
      </div>

      {/* File name - bottom center */}
      <div className="absolute bottom-6 left-0 right-0 text-center pointer-events-none">
        <span className="text-white/70 text-sm bg-black/50 px-3 py-1.5 rounded-full">
          {file.name}
        </span>
      </div>

    </div>
  );
};

export default LongPressPreviewModal;
