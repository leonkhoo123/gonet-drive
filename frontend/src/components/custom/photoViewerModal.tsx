import React, { useState, useCallback, useMemo, useEffect, useRef } from "react";
import { X, Download, ChevronLeft, ChevronRight, Loader2 } from "lucide-react";
import { type FileInterface, downloadFiles } from "@/api/api-file";
import { useDialogHistory } from "@/hooks/useDialogHistory";

interface PhotoViewerModalProps {
  initialFile: FileInterface | null;
  allItems?: FileInterface[];
  isOpen: boolean;
  onClose: () => void;
}

const SWIPE_THRESHOLD = 80; // px to trigger prev/next on swipe

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
  const swipeHandled = useRef(false);
  const touchStartX = useRef(0);
  const thumbStripRef = useRef<HTMLDivElement>(null);
  const isScrollingStrip = useRef(false);
  const scrollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleStripScroll = useCallback(() => {
    isScrollingStrip.current = true;
    if (scrollTimerRef.current) clearTimeout(scrollTimerRef.current);
    scrollTimerRef.current = setTimeout(() => {
      isScrollingStrip.current = false;
    }, 150);
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
    }
  }, [isOpen, initialIndex]);

  const currentFile = photoFiles[currentIndex] ?? initialFile;
  const hasMultiple = photoFiles.length > 1;
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === photoFiles.length - 1;

  // ---- Navigation (reset loading synchronously to avoid race) ----
  const goPrev = useCallback(() => {
    if (!isFirst) {
      setImageLoaded(false);
      setCurrentIndex((i) => i - 1);
    }
  }, [isFirst]);

  const goNext = useCallback(() => {
    if (!isLast) {
      setImageLoaded(false);
      setCurrentIndex((i) => i + 1);
    }
  }, [isLast]);

  const goTo = useCallback(
    (index: number) => {
      if (isScrollingStrip.current) return;
      if (index >= 0 && index < photoFiles.length) {
        setImageLoaded(false);
        setCurrentIndex(index);
      }
    },
    [photoFiles.length]
  );

  // Scroll active thumbnail to center when currentIndex changes.
  // Only suppress when user is actively dragging the strip — otherwise
  // always center (prev/next buttons, keyboard, tap).
  useEffect(() => {
    if (isScrollingStrip.current) return;
    const strip = thumbStripRef.current;
    if (!strip) return;
    const btn = strip.children[currentIndex] as HTMLElement | undefined;
    if (btn) {
      btn.scrollIntoView({ behavior: "auto", block: "nearest", inline: "center" });
    }
  }, [currentIndex]);

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

  // ---- Touch swipe ----
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
    swipeHandled.current = false;
  }, []);

  const handleTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      if (swipeHandled.current) return;
      const deltaX = e.changedTouches[0].clientX - touchStartX.current;
      if (Math.abs(deltaX) > SWIPE_THRESHOLD) {
        swipeHandled.current = true;
        if (deltaX > 0) {
          goPrev();
        } else {
          goNext();
        }
      }
    },
    [goPrev, goNext]
  );

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
    <div className="fixed inset-0 z-[100] bg-black/90 flex items-center justify-center select-none">
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

      {/* Left/Right hit zones — always clickable, arrow icon follows UI visibility + bounds */}
      {hasMultiple && (
        <>
          <button
            onClick={(e) => {
              e.stopPropagation();
              goPrev();
            }}
            className="flex absolute left-0 top-24 bottom-24 w-1/4 z-20 items-center justify-start pl-4 cursor-pointer"
            title="Previous"
          >
            <span
              className={`inline-flex items-center justify-center w-12 h-12 rounded-full bg-black/40 transition-opacity ${
                !showUI || isFirst ? "opacity-0" : "opacity-100"
              }`}
            >
              <ChevronLeft className="w-8 h-8 text-white drop-shadow-lg" />
            </span>
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              goNext();
            }}
            className="flex absolute right-0 top-24 bottom-24 w-1/4 z-20 items-center justify-end pr-4 cursor-pointer"
            title="Next"
          >
            <span
              className={`inline-flex items-center justify-center w-12 h-12 rounded-full bg-black/40 transition-opacity ${
                !showUI || isLast ? "opacity-0" : "opacity-100"
              }`}
            >
              <ChevronRight className="w-8 h-8 text-white drop-shadow-lg" />
            </span>
          </button>
        </>
      )}

      {/* Image Container — fills entire modal, overlays sit on top */}
      <div
        className="w-full h-full flex items-center justify-center relative"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
      >
        {/* Loading spinner — shown while image not yet loaded */}
        {!imageLoaded && (
          <div className="absolute inset-0 flex items-center justify-center z-10">
            <Loader2 className="w-12 h-12 text-white/70 animate-spin" />
          </div>
        )}

        <img
          key={currentFile.url}
          src={currentFile.url}
          alt={currentFile.name}
          className={`max-w-full max-h-full object-contain transition-opacity duration-300 ${
            imageLoaded ? "opacity-100" : "opacity-0"
          }`}
          onClick={handleImageClick}
          draggable={false}
          onLoad={() => { setImageLoaded(true); }}
        />
      </div>

      {/* Bottom Thumbnail Strip */}
      {hasMultiple && (
        <div
          className={`absolute bottom-0 left-0 right-0 z-20 pt-3 pb-4 ${uiHidden}`}
          style={{ paddingBottom: "max(1rem, env(safe-area-inset-bottom, 8px))" }}
        >
          <div
            ref={thumbStripRef}
            className="flex gap-3 overflow-x-auto scrollbar-hide py-1"
            onScroll={handleStripScroll}
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
