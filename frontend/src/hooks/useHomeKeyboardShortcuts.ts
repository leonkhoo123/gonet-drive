import { useEffect } from "react";
import { type FileInterface } from "@/api/api-file";

interface UseHomeKeyboardShortcutsProps {
  handleRefresh: () => Promise<void> | void;
  handleSelectAll: () => void;
  handleCopy: () => void;
  handleCut: () => void;
  handlePaste: () => Promise<void> | void;
  handleBack: () => void;
  isRenameDialogOpen: boolean;
  isCreateFolderDialogOpen: boolean;
  isDeleteDialogOpen: boolean;
  isPropertiesDialogOpen: boolean;
  isDownloadDirDialogOpen: boolean;
  selectedVideo: FileInterface | null;
  selectedPhoto: FileInterface | null;
}

export function useHomeKeyboardShortcuts({
  handleRefresh,
  handleSelectAll,
  handleCopy,
  handleCut,
  handlePaste,
  handleBack,
  isRenameDialogOpen,
  isCreateFolderDialogOpen,
  isDeleteDialogOpen,
  isPropertiesDialogOpen,
  isDownloadDirDialogOpen,
  selectedVideo,
  selectedPhoto,
}: UseHomeKeyboardShortcutsProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignore if typing in an input, textarea, etc.
      const target = e.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable
      ) {
        return;
      }

      // F5 or Ctrl/Cmd + R for refresh
      if (e.key === "F5" || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "r")) {
        e.preventDefault();
        void handleRefresh();
        return;
      }

      // Only handle these if no dialogs or video player are open
      if (
        isRenameDialogOpen ||
        isCreateFolderDialogOpen ||
        isDeleteDialogOpen ||
        isPropertiesDialogOpen ||
        isDownloadDirDialogOpen ||
        !!selectedVideo ||
        !!selectedPhoto
      ) {
        return;
      }

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") {
        e.preventDefault();
        handleSelectAll();
      } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "c") {
        e.preventDefault();
        handleCopy();
      } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "x") {
        e.preventDefault();
        handleCut();
      } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "v") {
        e.preventDefault();
        void handlePaste();
      } else if (e.key === "Backspace") {
        e.preventDefault();
        handleBack();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [
    handleRefresh,
    handleSelectAll,
    handleCopy,
    handleCut,
    handlePaste,
    handleBack,
    isRenameDialogOpen,
    isCreateFolderDialogOpen,
    isDeleteDialogOpen,
    isPropertiesDialogOpen,
    isDownloadDirDialogOpen,
    selectedVideo,
    selectedPhoto,
  ]);
}
