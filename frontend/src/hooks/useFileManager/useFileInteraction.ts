import { useCallback, useRef, useState } from "react";
import type { FileInterface } from "@/api/api-file";

interface UseFileInteractionProps {
  onFileClick: (fileInfo: FileInterface, index: number, event: React.MouseEvent) => void;
  onFileDoubleClick: (fileInfo: FileInterface) => void;
  onFileContextMenu: (fileInfo: FileInterface, index: number) => void;
  selectedItems: Set<string>;
  isTouchDevice: boolean;
}

export function useFileInteraction({
  onFileClick,
  onFileDoubleClick,
  onFileContextMenu,
  selectedItems,
  isTouchDevice,
}: UseFileInteractionProps) {
  const touchTimer = useRef<number | null>(null);
  const [transitioningFolder, setTransitioningFolder] = useState<string | null>(null);

  const handleTouchStart = useCallback((file: FileInterface, index: number) => {
    if (touchTimer.current) {
      window.clearTimeout(touchTimer.current);
    }
    // Set a timer for 500ms to trigger right-click (context menu) behavior
    touchTimer.current = window.setTimeout(() => {
      // Trigger the selection of the item, so context menu actions apply to it
      onFileContextMenu(file, index);
      
      // Dispatch a context menu event on the element to open the Radix ContextMenu
      const el = document.getElementById(`file-item-${index}`);
      if (el) {
        // Prevent default tap behavior after a long press
        // This stops the tap from triggering a normal click
        el.setAttribute('data-long-pressed', 'true');
        window.setTimeout(() => {
          el.removeAttribute('data-long-pressed');
        }, 300);
        
        if (window.matchMedia("(pointer: coarse)").matches) {
          return;
        }

        el.dispatchEvent(new MouseEvent('contextmenu', {
          bubbles: true,
          cancelable: true,
          clientX: el.getBoundingClientRect().left + 20,
          clientY: el.getBoundingClientRect().top + 20
        }));
      }
    }, 500);
  }, [onFileContextMenu]);

  const handleTouchEnd = useCallback(() => {
    if (touchTimer.current) {
      window.clearTimeout(touchTimer.current);
      touchTimer.current = null;
    }
  }, []);

  const handleTouchMove = useCallback(() => {
    // If the user scrolls or moves their finger, cancel the long press
    if (touchTimer.current) {
      window.clearTimeout(touchTimer.current);
      touchTimer.current = null;
    }
  }, []);

  const handleItemClick = useCallback((file: FileInterface, index: number, e: React.MouseEvent) => {
    const el = e.currentTarget;
    if (el.getAttribute('data-long-pressed') === 'true') {
      e.preventDefault();
      e.stopPropagation();
      return;
    }

    if (isTouchDevice && selectedItems.size === 0 && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
      // Delay for animation before actually doing the action
      setTransitioningFolder(file.name);
      
      e.stopPropagation();
      e.preventDefault();

      const safeEvent = { 
        stopPropagation: () => { /* intentionally empty */ }, 
        preventDefault: () => { /* intentionally empty */ },
        shiftKey: e.shiftKey,
        ctrlKey: e.ctrlKey,
        metaKey: e.metaKey,
      } as unknown as React.MouseEvent;
      
      setTimeout(() => {
        onFileClick(file, index, safeEvent);
        if (file.type !== "dir") {
          setTimeout(() => {
            setTransitioningFolder((prev) => prev === file.name ? null : prev);
          }, 500);
        }
      }, 75);
      return;
    }
    
    onFileClick(file, index, e);
  }, [isTouchDevice, onFileClick, selectedItems.size]);

  const handleItemDoubleClick = useCallback((file: FileInterface) => {
    if (!isTouchDevice) {
      setTransitioningFolder(file.name);
      setTimeout(() => {
        onFileDoubleClick(file);
        if (file.type !== "dir") {
          setTimeout(() => {
            setTransitioningFolder((prev) => prev === file.name ? null : prev);
          }, 500);
        }
      }, 75);
      return;
    }
    onFileDoubleClick(file);
  }, [isTouchDevice, onFileDoubleClick]);

  return {
    handleTouchStart,
    handleTouchEnd,
    handleTouchMove,
    handleItemClick,
    handleItemDoubleClick,
    transitioningFolder,
    setTransitioningFolder,
  };
}
