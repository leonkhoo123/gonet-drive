import React, { useState, useCallback, useMemo, useEffect, useRef } from "react";
import { X, Download, ChevronLeft, ChevronRight, Loader2 } from "lucide-react";
import { type FileInterface, downloadFiles } from "@/api/api-file";
import { useDialogHistory } from "@/hooks/useDialogHistory";

const HIT_FLASH_DURATION = 300; // ms
const CLOSE_ANIM_DURATION = 200; // ms

interface PhotoViewerModalProps {
  initialFile: FileInterface | null;
  allItems?: FileInterface[];
  isOpen: boolean;
  onClose: () => void;
}

const SWIPE_THRESHOLD = 80; // px to trigger prev/next on swipe
const SWIPE_CLOSE_THRESHOLD = 100; // px vertical to trigger close
const SWIPE_CLOSE_VELOCITY = 0.5; // px/ms flick threshold

const thumbUrl = (f: FileInterface) => f.url.replace("/photo/play/", "/photo/thumbnail/");

const ThumbnailItem = React.memo(
  ({
    file,
    isActive,
    index,
    onGoTo,
  }: {
    file: FileInterface;
    isActive: boolean;
    index: number;
    onGoTo: (idx: number) => void;
  }) => (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onGoTo(index);
      }}
      className={`flex-shrink-0 h-16 transition-all focus:outline-none ${
        isActive
          ? "scale-105"
          : "opacity-60 hover:opacity-100"
      }`}
    >
      <img
        src={thumbUrl(file)}
        alt={file.name}
        className={`h-full w-auto rounded-md ${
          isActive
            ? "ring-2 ring-white ring-offset-1 ring-offset-transparent"
            : ""
        }`}
        loading="lazy"
      />
    </button>
  )
);

ThumbnailItem.displayName = "ThumbnailItem";

export const PhotoViewerModal: React.FC<PhotoViewerModalProps> = ({
  initialFile,
  allItems,
  isOpen,
  onClose,
}) => {
  const [showUI, setShowUI] = useState(true);
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [isClosing, setIsClosing] = useState(false); // swipe-down close animation state
  const closeAnimRef = useRef(false); // prevent double-close during animation
  const imgRef = useRef<HTMLImageElement>(null);
  const swipeHandled = useRef(false);
  const touchStartX = useRef(0);
  const touchStartY = useRef(0);
  const touchStartTime = useRef(0);
  const thumbStripRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const isScrollingStrip = useRef(false);
  const scrollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isProgrammaticScroll = useRef(false);
  const leftHitRef = useRef<HTMLButtonElement>(null);
  const rightHitRef = useRef<HTMLButtonElement>(null);

  const handleStripScroll = useCallback(() => {
    // Ignore scroll events triggered by our own scrollIntoView calls.
    if (isProgrammaticScroll.current) {
      isProgrammaticScroll.current = false;
      return;
    }
    isScrollingStrip.current = true;
    if (scrollTimerRef.current) clearTimeout(scrollTimerRef.current);
    scrollTimerRef.current = setTimeout(() => {
      isScrollingStrip.current = false;
    }, 150);
  }, []);

  // Translate vertical mouse-wheel into horizontal scroll for the thumbnail strip.
  const handleStripWheel = useCallback((e: React.WheelEvent) => {
    const strip = thumbStripRef.current;
    if (!strip) return;
    // Only intercept if there is horizontal overflow to scroll
    if (strip.scrollWidth <= strip.clientWidth) return;
    e.preventDefault();
    strip.scrollLeft += e.deltaY;
  }, []);

  // Build ordered photo list from file-list order
  const photoFiles = useMemo(() => {
    if (!allItems || allItems.length === 0) {
      return initialFile ? [initialFile] : [];
    }
    const photos = allItems.filter((item) => item.media_type === "photo");
    return photos.length > 0 ? photos : initialFile ? [initialFile] : [];
  }, [allItems, initialFile]);

  // Find current index by matching path, fallback to name
  const initialIndex = useMemo(() => {
    if (!initialFile) return 0;
    const idx = photoFiles.findIndex(
      (f) => f.path === initialFile.path || f.name === initialFile.name
    );
    return idx >= 0 ? idx : 0;
    // Only recompute when initialFile changes (modal opens)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialFile, isOpen]);

  const [currentIndex, setCurrentIndex] = useState(initialIndex);

  // Reset when modal opens
  useEffect(() => {
    if (isOpen) {
      setCurrentIndex(initialIndex);
      setShowUI(true);
      setImageLoaded(false);
      setImageError(false);
      setIsClosing(false);
      closeAnimRef.current = false;
    }
  }, [isOpen, initialIndex]);

  const currentFile = photoFiles[currentIndex] ?? initialFile;

  // Guard against cached images where onLoad fires before React binds the handler.
  // HTMLImageElement.complete is true when the image finished loading (success or error),
  // regardless of whether anyone listened to the load event.
  useEffect(() => {
    if (imgRef.current?.complete) {
      setImageLoaded(true);
    }
  }, [currentFile.url]);

  // Timeout fallback: if image never loads and never errors, force-show after 10s
  // so the user is not stuck with an infinite spinner.
  useEffect(() => {
    const timer = setTimeout(() => {
      setImageLoaded((prev) => {
        if (!prev && !imageError) return true;
        return prev;
      });
    }, 10_000);
    return () => { clearTimeout(timer); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentFile.url]);
  const hasMultiple = photoFiles.length > 1;
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === photoFiles.length - 1;

  // ---- Hit flash (Web Animations API — survives React reconciliation) ----
  const triggerHitFlash = useCallback((side: "left" | "right") => {
    const el = side === "left" ? leftHitRef.current : rightHitRef.current;
    if (!el) return;
    el.animate(
      [
        { backgroundColor: "rgba(255,255,255,0)" },
        { backgroundColor: "rgba(255,255,255,0.3)" },
        { backgroundColor: "rgba(255,255,255,0)" },
      ],
      { duration: HIT_FLASH_DURATION, easing: "ease-out" }
    );
  }, []);

  // ---- Navigation (reset loading synchronously to avoid race) ----
  const goPrev = useCallback(() => {
    if (!isFirst) {
      setImageLoaded(false);
      setImageError(false);
      setCurrentIndex((i) => i - 1);
    }
  }, [isFirst]);

  const goNext = useCallback(() => {
    if (!isLast) {
      setImageLoaded(false);
      setImageError(false);
      setCurrentIndex((i) => i + 1);
    }
  }, [isLast]);

  const goTo = useCallback(
    (index: number) => {
      if (isScrollingStrip.current) return;
      if (index >= 0 && index < photoFiles.length) {
        setImageLoaded(false);
        setImageError(false);
        setCurrentIndex(index);
      }
    },
    [photoFiles.length]
  );

  // Scroll active thumbnail to center when currentIndex changes.
  useEffect(() => {
    if (isScrollingStrip.current) return;
    const strip = thumbStripRef.current;
    if (!strip) return;
    const btn = strip.children[currentIndex] as HTMLElement | undefined;
    if (btn) {
      isProgrammaticScroll.current = true;
      btn.scrollIntoView({ behavior: "auto", block: "nearest", inline: "center" });
    }
  }, [currentIndex, showUI]);

  // ---- Keyboard ----
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isOpen) return;
      if (e.key === "Escape") {
        e.stopPropagation();
        e.preventDefault();
        onClose();
        return;
      }
      if (e.key === "ArrowLeft") {
        e.stopPropagation();
        e.preventDefault();
        goPrev();
        return;
      }
      if (e.key === "ArrowRight") {
        e.stopPropagation();
        e.preventDefault();
        goNext();
      }
    },
    [isOpen, onClose, goPrev, goNext]
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener("keydown", handleKeyDown, { capture: true });
    };
  }, [handleKeyDown]);

  useDialogHistory(isOpen, onClose);

  // ---- Swipe-down close (mobile) ----
  const triggerSwipeClose = useCallback(() => {
    if (closeAnimRef.current) return;
    closeAnimRef.current = true;
    setIsClosing(true);
    setTimeout(() => {
      onClose();
    }, CLOSE_ANIM_DURATION);
  }, [onClose]);

  // ---- Touch swipe (horizontal: prev/next; vertical-down: close on mobile) ----
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
    touchStartY.current = e.touches[0].clientY;
    touchStartTime.current = Date.now();
    swipeHandled.current = false;
  }, []);

  const handleTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      if (swipeHandled.current) return;
      const deltaX = e.changedTouches[0].clientX - touchStartX.current;
      const deltaY = e.changedTouches[0].clientY - touchStartY.current;
      const elapsed = Date.now() - touchStartTime.current;

      // Vertical swipe-down close (mobile only)
      if (deltaY > SWIPE_CLOSE_THRESHOLD && Math.abs(deltaY) > Math.abs(deltaX)) {
        // Distance threshold met
        swipeHandled.current = true;
        triggerSwipeClose();
        return;
      }
      if (deltaY > 0 && elapsed > 0 && deltaY / elapsed > SWIPE_CLOSE_VELOCITY && Math.abs(deltaY) > Math.abs(deltaX)) {
        // Velocity threshold met (flick)
        swipeHandled.current = true;
        triggerSwipeClose();
        return;
      }

      // Horizontal swipe prev/next
      if (Math.abs(deltaX) > SWIPE_THRESHOLD) {
        swipeHandled.current = true;
        if (deltaX > 0) {
          goPrev();
        } else {
          goNext();
        }
      }
    },
    [goPrev, goNext, triggerSwipeClose]
  );

  // ---- Backdrop click: desktop only (md+) ----
  const handleBackdropClick = useCallback(() => {
    // On mobile (< md), backdrop click does nothing; close via swipe-down or X button
    if (typeof window !== "undefined" && window.innerWidth >= 768) {
      onClose();
    }
  }, [onClose]);

  // ---- Image click: instant UI toggle ----
  const handleImageClick = useCallback(
    (e: React.MouseEvent) => {
      if (swipeHandled.current) {
        swipeHandled.current = false;
        return;
      }
      e.stopPropagation();
      setShowUI((prev) => !prev);
    },
    []
  );

  if (!isOpen || !initialFile) return null;

  const uiHidden = showUI ? "" : "hidden";

  return (
    <div
      ref={containerRef}
      className={`fixed inset-0 z-[100] bg-black/90 flex items-center justify-center select-none transition-transform duration-200 ease-out ${
        isClosing ? "translate-y-full" : "translate-y-0"
      }`}
      onClick={handleBackdropClick}
    >
      {/* Top Bar */}
      <div
        className={`absolute top-0 left-0 right-0 p-4 flex justify-between items-center bg-gradient-to-b from-black/60 to-transparent z-10 text-white ${uiHidden}`}
      >
        <div className="flex items-center gap-2 max-w-[70%] overflow-x-auto scrollbar-hide touch-pan-x">
          <span className="font-medium whitespace-nowrap">{currentFile.name}</span>
          {hasMultiple && (
            <span className="text-white/60 text-sm whitespace-nowrap">
              {currentIndex + 1} / {photoFiles.length}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              downloadFiles([currentFile.path]);
            }}
            className="p-2 hover:bg-white/20 rounded-full transition-colors"
            title="Download"
          >
            <Download className="w-6 h-6" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onClose();
            }}
            className="p-2 hover:bg-white/20 rounded-full transition-colors"
            title="Close"
          >
            <X className="w-6 h-6" />
          </button>
        </div>
      </div>

      {/* Mobile: wide hit zones w-1/5 for easy thumb navigation */}
      {hasMultiple && (
        <>
          {/* Mobile left hit zone */}
          <button
            ref={leftHitRef}
            onClick={(e) => {
              e.stopPropagation();
              triggerHitFlash("left");
              goPrev();
            }}
            className="flex md:hidden absolute left-0 top-24 bottom-24 w-1/5 z-20 items-center justify-start pl-4 cursor-pointer rounded-r-xl transition-colors"
            title="Previous"
          >
            <ChevronLeft
              className={`w-8 h-8 text-white drop-shadow-lg transition-opacity ${
                !showUI || isFirst ? "opacity-0" : "opacity-100"
              }`}
            />
          </button>
          {/* Mobile right hit zone */}
          <button
            ref={rightHitRef}
            onClick={(e) => {
              e.stopPropagation();
              triggerHitFlash("right");
              goNext();
            }}
            className="flex md:hidden absolute right-0 top-24 bottom-24 w-1/5 z-20 items-center justify-end pr-4 cursor-pointer rounded-l-xl transition-colors"
            title="Next"
          >
            <ChevronRight
              className={`w-8 h-8 text-white drop-shadow-lg transition-opacity ${
                !showUI || isLast ? "opacity-0" : "opacity-100"
              }`}
            />
          </button>

          {/* Desktop: precise arrow buttons (no wide hit area) */}
          <button
            onClick={(e) => {
              e.stopPropagation();
              goPrev();
            }}
            className="hidden md:flex absolute left-4 top-1/2 -translate-y-1/2 z-20 items-center justify-center w-12 h-12 rounded-full bg-black/40 hover:bg-black/60 transition-colors cursor-pointer"
            title="Previous"
            style={{ opacity: !showUI || isFirst ? 0 : 1 }}
          >
            <ChevronLeft className="w-8 h-8 text-white drop-shadow-lg" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              goNext();
            }}
            className="hidden md:flex absolute right-4 top-1/2 -translate-y-1/2 z-20 items-center justify-center w-12 h-12 rounded-full bg-black/40 hover:bg-black/60 transition-colors cursor-pointer"
            title="Next"
            style={{ opacity: !showUI || isLast ? 0 : 1 }}
          >
            <ChevronRight className="w-8 h-8 text-white drop-shadow-lg" />
          </button>
        </>
      )}

      {/* Image Container — fills entire modal, overlays sit on top */}
      <div
        className="w-full h-full flex items-center justify-center relative"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
      >
        {/* Loading spinner — pointer-events-none so clicks pass through to close/prev/next */}
        {!imageLoaded && !imageError && (
          <div className="absolute inset-0 flex items-center justify-center z-10 pointer-events-none">
            <Loader2 className="w-12 h-12 text-white/70 animate-spin" />
          </div>
        )}

        {/* Error message */}
        {imageError && (
          <div className="absolute inset-0 flex items-center justify-center z-10 pointer-events-none">
            <span className="text-white/50 text-sm">Failed to load image</span>
          </div>
        )}

        <img
          ref={imgRef}
          key={currentFile.url}
          src={currentFile.url}
          alt={currentFile.name}
          className={`max-w-full max-h-full object-contain transition-opacity duration-300 ${
            imageLoaded ? "opacity-100" : "opacity-0"
          }`}
          onClick={handleImageClick}
          draggable={false}
          onLoad={() => { setImageLoaded(true); }}
          onError={() => { setImageError(true); }}
        />
      </div>

      {/* Bottom Thumbnail Strip */}
      {hasMultiple && (
        <div
          className={`absolute bottom-0 left-0 right-0 z-20 pt-3 pb-4 ${uiHidden}`}
          style={{ paddingBottom: "max(1rem, env(safe-area-inset-bottom, 8px))" }}
          onClick={(e) => { e.stopPropagation(); }}
        >
          <div
            ref={thumbStripRef}
            className="flex gap-3 overflow-x-auto scrollbar-hide py-1"
            onScroll={handleStripScroll}
            onWheel={handleStripWheel}
            style={{
              paddingLeft: "calc(50% - 2rem)",
              paddingRight: "calc(50% - 2rem)",
              touchAction: "pan-x",
            }}
          >
            {photoFiles.map((file, idx) => (
              <ThumbnailItem
                key={file.path}
                file={file}
                isActive={idx === currentIndex}
                index={idx}
                onGoTo={goTo}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default PhotoViewerModal;
