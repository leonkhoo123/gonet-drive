import React, { useState, useCallback } from "react";
import { X, Loader2, Download } from "lucide-react";
import { type FileInterface, downloadFiles } from "@/api/api-file";
import { useDialogHistory } from "@/hooks/useDialogHistory";

interface PhotoViewerModalProps {
  initialFile: FileInterface | null;
  allItems?: FileInterface[];
  isOpen: boolean;
  onClose: () => void;
}

export const PhotoViewerModal: React.FC<PhotoViewerModalProps> = ({
  initialFile,
  isOpen,
  onClose,
}) => {
  const [isLoading, setIsLoading] = useState(true);
  const imgRef = React.useRef<HTMLImageElement>(null);

  // Reset loading state when file changes
  React.useEffect(() => {
    if (initialFile) {
      setIsLoading(true);
      if (imgRef.current?.complete) {
        setIsLoading(false);
      }
    }
  }, [initialFile]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isOpen) return;
      if (e.key === "Escape") {
        e.stopPropagation();
        e.preventDefault();
        onClose();
      }
    },
    [isOpen, onClose]
  );

  React.useEffect(() => {
    window.addEventListener("keydown", handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener("keydown", handleKeyDown, { capture: true });
    };
  }, [handleKeyDown]);

  useDialogHistory(isOpen, onClose);

  if (!isOpen || !initialFile) return null;

  return (
    <div className="fixed inset-0 z-[100] bg-black/90 flex items-center justify-center select-none">
      {/* Top Bar */}
      <div className="absolute top-0 left-0 right-0 p-4 flex justify-between items-center bg-gradient-to-b from-black/60 to-transparent z-10 text-white">
        <div className="flex items-center gap-2 max-w-[70%] overflow-x-auto scrollbar-hide touch-pan-x">
          <span className="font-medium whitespace-nowrap">{initialFile.name}</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              downloadFiles([initialFile.path]);
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

      {/* Image Container */}
      <div 
        className="w-full h-full flex items-center justify-center p-4 relative touch-none"
        onClick={onClose} // Clicking background closes
      >
        {isLoading && (
          <div className="absolute inset-0 flex items-center justify-center z-0">
            <div className="animate-spin text-white/50" style={{ animationDuration: '1.5s' }}>
              <Loader2 className="w-12 h-12" />
            </div>
          </div>
        )}
        
        {/* We stop propagation here so clicking the image itself doesn't close the modal */}
        <img
          ref={imgRef}
          src={initialFile.url}
          alt={initialFile.name}
          className={`max-w-full max-h-[90dvh] object-contain transition-opacity duration-300 z-10 ${isLoading ? 'opacity-0' : 'opacity-100'}`}
          onClick={(e) => { e.stopPropagation(); }}
          onLoad={() => { setIsLoading(false); }}
          onError={() => { setIsLoading(false); }}
        />
      </div>
    </div>
  );
};

export default PhotoViewerModal;