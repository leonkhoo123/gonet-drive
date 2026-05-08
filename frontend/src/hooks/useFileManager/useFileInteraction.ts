import { useCallback, useRef, useState, useEffect } from "react";
import type { FileInterface } from "@/api/api-file";

interface UseFileInteractionProps {
  onFileClick: (fileInfo: FileInterface, index: number, event: React.MouseEvent) => void;
  onFileDoubleClick: (fileInfo: FileInterface) => void;
  onFileContextMenu: (fileInfo: FileInterface, index: number) => void;
  selectedItems: Set<string>;
  isTouchDevice: boolean;
  // Drag-select on mobile
  onDragSelectStart?: (file: FileInterface, index: number) => void;
  onDragSelectItem?: (index: number) => void;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}

const LONG_PRESS_MS = 500;
const DRAG_THRESHOLD_PX = 10;
const EDGE_SCROLL_ZONE = 40;
const MAX_SCROLL_SPEED = 12;

export function useFileInteraction({
  onFileClick,
  onFileDoubleClick,
  onFileContextMenu,
  selectedItems,
  isTouchDevice,
  onDragSelectStart,
  onDragSelectItem,
  scrollContainerRef,
}: UseFileInteractionProps) {
  const touchTimer = useRef<number | null>(null);
  const longPressActive = useRef(false);
  const touchStartPos = useRef<{ x: number; y: number } | null>(null);
  const dragSelectActive = useRef(false);
  const longPressedFile = useRef<{ file: FileInterface; index: number } | null>(null);
  const lastDragSelectedIndex = useRef<number | null>(null);
  const autoScrollRaf = useRef<number | null>(null);
  const touchClientY = useRef<number>(0);
  const touchClientX = useRef<number>(0);

  const [transitioningFolder, setTransitioningFolder] = useState<string | null>(null);

  // Stash latest callbacks so native listeners don't need re-binding
  const onDragSelectItemRef = useRef(onDragSelectItem);
  onDragSelectItemRef.current = onDragSelectItem;

  const stopAutoScroll = useCallback(() => {
    if (autoScrollRaf.current !== null) {
      cancelAnimationFrame(autoScrollRaf.current);
      autoScrollRaf.current = null;
    }
  }, []);

  // ---- Hit-test helper: find file index under current finger position ----
  const hitTestAndSelect = useCallback(() => {
    const hitEl = document.elementFromPoint(touchClientX.current, touchClientY.current);
    if (hitEl) {
      const itemEl = hitEl.closest('[data-file-index]');
      if (itemEl) {
        const idx = parseInt(itemEl.getAttribute('data-file-index') ?? '', 10);
        if (!isNaN(idx) && idx !== lastDragSelectedIndex.current) {
          lastDragSelectedIndex.current = idx;
          onDragSelectItemRef.current?.(idx);
        }
      }
    }
  }, []);

  // ---- Auto-scroll RAF loop ----
  const startAutoScroll = useCallback(() => {
    if (autoScrollRaf.current !== null) return; // already running

    const scroll = () => {
      const c = scrollContainerRef?.current;
      if (!c || !dragSelectActive.current) {
        autoScrollRaf.current = null;
        return;
      }

      const r = c.getBoundingClientRect();
      const ry = touchClientY.current - r.top;

      let inZone = false;
      if (ry >= 0 && ry < EDGE_SCROLL_ZONE) {
        const speed = ((EDGE_SCROLL_ZONE - ry) / EDGE_SCROLL_ZONE) * MAX_SCROLL_SPEED;
        c.scrollTop = Math.max(0, c.scrollTop - speed);
        inZone = true;
      } else if (ry > r.height - EDGE_SCROLL_ZONE && ry <= r.height) {
        const speed = ((ry - (r.height - EDGE_SCROLL_ZONE)) / EDGE_SCROLL_ZONE) * MAX_SCROLL_SPEED;
        c.scrollTop = Math.min(c.scrollHeight - c.clientHeight, c.scrollTop + speed);
        inZone = true;
      }

      if (inZone) {
        // Re-check hit target while auto-scrolling (finger may be stationary)
        hitTestAndSelect();
        autoScrollRaf.current = requestAnimationFrame(scroll);
      } else {        // Finger left edge zone — stop
        autoScrollRaf.current = null;
      }
    };

    autoScrollRaf.current = requestAnimationFrame(scroll);
  }, [scrollContainerRef, hitTestAndSelect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (touchTimer.current) window.clearTimeout(touchTimer.current);
      stopAutoScroll();
    };
  }, [stopAutoScroll]);

  // Called from individual item's onTouchStart (React)
  const handleTouchStart = useCallback((file: FileInterface, index: number, e: React.TouchEvent) => {
    if (touchTimer.current) window.clearTimeout(touchTimer.current);
    stopAutoScroll();
    longPressActive.current = false;
    dragSelectActive.current = false;
    longPressedFile.current = null;
    lastDragSelectedIndex.current = null;

    const touch = e.nativeEvent.touches[0];
    touchStartPos.current = { x: touch.clientX, y: touch.clientY };
    touchClientY.current = touch.clientY;
    touchClientX.current = touch.clientX;

    touchTimer.current = window.setTimeout(() => {
      longPressActive.current = true;
      longPressedFile.current = { file, index };

      // Mark element to prevent subsequent click
      const el = document.getElementById(`file-item-${index}`);
      if (el) {
        el.setAttribute('data-long-pressed', 'true');
        window.setTimeout(() => {
          el.removeAttribute('data-long-pressed');
        }, 500);
      }

      onDragSelectStart?.(file, index);
      lastDragSelectedIndex.current = index;
    }, LONG_PRESS_MS);
  }, [stopAutoScroll, onDragSelectStart]);

  // ---- Native touchmove on scroll container (passive: false) ----
  const handleTouchMoveNative = useCallback((e: TouchEvent) => {
    const touch = e.touches[0];
    touchClientY.current = touch.clientY;
    touchClientX.current = touch.clientX;

    if (!longPressActive.current) {
      // Cancel long-press timer if finger moved significantly
      if (touchStartPos.current) {
        const dx = Math.abs(touch.clientX - touchStartPos.current.x);
        const dy = Math.abs(touch.clientY - touchStartPos.current.y);
        if (dx > DRAG_THRESHOLD_PX || dy > DRAG_THRESHOLD_PX) {
          if (touchTimer.current) {
            window.clearTimeout(touchTimer.current);
            touchTimer.current = null;
          }
        }
      }
      return;
    }

    // Check if moved enough from long-press origin to enter drag-select
    if (!dragSelectActive.current && touchStartPos.current) {
      const dx = Math.abs(touch.clientX - touchStartPos.current.x);
      const dy = Math.abs(touch.clientY - touchStartPos.current.y);
      if (dx > DRAG_THRESHOLD_PX || dy > DRAG_THRESHOLD_PX) {
        dragSelectActive.current = true;
        // Block browser from handling any touch gestures on the scroll container
        const c = scrollContainerRef?.current;
        if (c) c.style.touchAction = 'none';
      } else {
        return; // Not enough movement yet
      }
    }

    if (!dragSelectActive.current) return;

    // Block browser scroll while drag-selecting
    e.preventDefault();

    // Hit-test current finger position
    hitTestAndSelect();

    // Start/restart auto-scroll if finger in edge zone
    const c = scrollContainerRef?.current;
    if (c) {
      const r = c.getBoundingClientRect();
      const ry = touch.clientY - r.top;
      const inZone =
        (ry >= 0 && ry < EDGE_SCROLL_ZONE) ||
        (ry > r.height - EDGE_SCROLL_ZONE && ry <= r.height);

      if (inZone) {
        startAutoScroll();
      } else {
        stopAutoScroll();
      }
    }
  }, [hitTestAndSelect, startAutoScroll, stopAutoScroll, scrollContainerRef]);

  // ---- Native touchend on scroll container ----
  const handleTouchEndNative = useCallback(() => {
    stopAutoScroll();

    // Restore browser touch handling
    if (dragSelectActive.current) {
      const c = scrollContainerRef?.current;
      if (c) c.style.touchAction = '';
    }

    if (touchTimer.current) {
      window.clearTimeout(touchTimer.current);
      touchTimer.current = null;
    }

    // If long press fired but finger stayed still → context menu
    if (longPressActive.current && !dragSelectActive.current) {
      const lp = longPressedFile.current;
      if (lp) {
        onFileContextMenu(lp.file, lp.index);

        const el = document.getElementById(`file-item-${lp.index}`);
        if (el && !window.matchMedia("(pointer: coarse)").matches) {
          el.dispatchEvent(new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: el.getBoundingClientRect().left + 20,
            clientY: el.getBoundingClientRect().top + 20
          }));
        }
      }
    }

    longPressActive.current = false;
    dragSelectActive.current = false;
    longPressedFile.current = null;
    lastDragSelectedIndex.current = null;
    touchStartPos.current = null;
  }, [stopAutoScroll, onFileContextMenu, scrollContainerRef]);

  // Attach native passive:false touchmove + touchend/touchcancel to scroll container
  useEffect(() => {
    const container = scrollContainerRef?.current;
    if (!container || !isTouchDevice) return;

    const onMove = (e: TouchEvent) => { handleTouchMoveNative(e); };
    const onEnd = () => { handleTouchEndNative(); };

    container.addEventListener('touchmove', onMove, { passive: false });
    container.addEventListener('touchend', onEnd);
    container.addEventListener('touchcancel', onEnd);

    return () => {
      container.removeEventListener('touchmove', onMove);
      container.removeEventListener('touchend', onEnd);
      container.removeEventListener('touchcancel', onEnd);
    };
  }, [scrollContainerRef?.current, isTouchDevice, handleTouchMoveNative, handleTouchEndNative]);

  // React-level touch handlers on items — cancel long press on any move/end
  const handleTouchMove = useCallback(() => {
    if (touchTimer.current) {
      window.clearTimeout(touchTimer.current);
      touchTimer.current = null;
    }
  }, []);

  const handleTouchEnd = useCallback(() => {
    if (touchTimer.current) {
      window.clearTimeout(touchTimer.current);
      touchTimer.current = null;
    }
  }, []);

  const handleItemClick = useCallback((file: FileInterface, index: number, e: React.MouseEvent) => {
    const el = e.currentTarget as HTMLElement;
    if (el.getAttribute('data-long-pressed') === 'true') {
      e.preventDefault();
      e.stopPropagation();
      return;
    }

    if (isTouchDevice && selectedItems.size === 0 && !e.shiftKey && !e.ctrlKey && !e.metaKey) {
      setTransitioningFolder(file.name);

      e.stopPropagation();
      e.preventDefault();

      const safeEvent = {
        stopPropagation: () => { /* noop */ },
        preventDefault: () => { /* noop */ },
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
