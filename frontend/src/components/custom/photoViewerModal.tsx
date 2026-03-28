import React, { useState, useEffect, useCallback, useMemo } from "react";
import { X, ChevronLeft, ChevronRight } from "lucide-react";
import type { FileInterface } from "@/api/api-file";
import { useDialogHistory } from "@/hooks/useDialogHistory";

interface PhotoViewerModalProps {
  initialFile: FileInterface | null;
  allItems: FileInterface[];
  isOpen: boolean;
  onClose: () => void;
}

export const PhotoViewerModal: React.FC<PhotoViewerModalProps> = ({
  initialFile,
  allItems,
  isOpen,
  onClose,
}) => {
  const images = useMemo(() => {
    return allItems.filter(item => item.type === 'file' && (item.media_type === "photo"));
  }, [allItems]);

  const [currentIndex, setCurrentIndex] = useState<number>(() => {
    if (initialFile) {
      const idx = allItems.filter(item => item.type === 'file' && item.media_type === "photo").findIndex(img => img.name === initialFile.name);
      return idx !== -1 ? idx : 0;
    }
    return 0;
  });

  useEffect(() => {
    if (initialFile && isOpen) {
      const index = images.findIndex(img => img.name === initialFile.name);
      if (index !== -1) {
        setCurrentIndex(index);
      }
    }
  }, [initialFile, isOpen, images]);

  const handleNext = useCallback(() => {
    setCurrentIndex((prev) => (prev + 1) % images.length);
  }, [images.length]);

  const handlePrev = useCallback(() => {
    setCurrentIndex((prev) => (prev - 1 + images.length) % images.length);
  }, [images.length]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isOpen) return;
      
      switch (e.key) {
        case "ArrowRight":
          e.preventDefault();
          handleNext();
          break;
        case "ArrowLeft":
          e.preventDefault();
          handlePrev();
          break;
        case "Escape":
          e.stopPropagation();
          e.preventDefault();
          onClose();
          break;
      }
    },
    [isOpen, handleNext, handlePrev, onClose]
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener("keydown", handleKeyDown, { capture: true });
    };
  }, [handleKeyDown]);

  useDialogHistory(isOpen, onClose);

  if (!isOpen || images.length === 0) return null;

  const currentImage = images[currentIndex];

  return (
    <div className="fixed inset-0 z-[100] bg-black/90 flex items-center justify-center select-none touch-none">
      {/* Top Bar */}
      <div className="absolute top-0 left-0 right-0 p-4 flex justify-between items-center bg-gradient-to-b from-black/60 to-transparent z-10 text-white">
        <div className="flex items-center gap-2 max-w-[80%]">
          <span className="font-medium truncate">{currentImage.name}</span>
          <span className="text-white/60 text-sm whitespace-nowrap">
            ({currentIndex + 1} / {images.length})
          </span>
        </div>
        <button
          onClick={onClose}
          className="p-2 hover:bg-white/20 rounded-full transition-colors"
        >
          <X className="w-6 h-6" />
        </button>
      </div>

      {/* Navigation Buttons (Desktop) */}
      {images.length > 1 && (
        <>
          <button
            onClick={(e) => {
              e.stopPropagation();
              handlePrev();
            }}
            className="absolute left-4 top-1/2 -translate-y-1/2 p-3 bg-black/40 hover:bg-black/70 text-white rounded-full transition-colors z-10 hidden md:block"
          >
            <ChevronLeft className="w-8 h-8" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleNext();
            }}
            className="absolute right-4 top-1/2 -translate-y-1/2 p-3 bg-black/40 hover:bg-black/70 text-white rounded-full transition-colors z-10 hidden md:block"
          >
            <ChevronRight className="w-8 h-8" />
          </button>
        </>
      )}

      {/* Image Container */}
      <div 
        className="w-full h-full flex items-center justify-center p-4 relative"
        onClick={onClose} // Clicking background closes
      >
        {/* We stop propagation here so clicking the image itself doesn't close the modal */}
        <img
          src={currentImage.url}
          alt={currentImage.name}
          className="max-w-full max-h-[90dvh] object-contain transition-opacity duration-300"
          onClick={(e) => { e.stopPropagation(); }}
        />
        
        {/* Touch areas for mobile navigation */}
        {images.length > 1 && (
          <>
            <div 
              className="absolute left-0 top-0 bottom-0 w-1/3 md:hidden z-[5]"
              onClick={(e) => {
                e.stopPropagation();
                handlePrev();
              }}
            />
            <div 
              className="absolute right-0 top-0 bottom-0 w-1/3 md:hidden z-[5]"
              onClick={(e) => {
                e.stopPropagation();
                handleNext();
              }}
            />
          </>
        )}
      </div>
    </div>
  );
};

export default PhotoViewerModal;