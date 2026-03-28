import { useEffect } from 'react';
import { type ItemsResponse, type FileInterface } from "@/api/api-file";

export function useKeyboardShortcuts({
  items,
  selectedItems,
  setSelectedItems,
  lastSelectedIndex,
  setLastSelectedIndex,
  selectionAnchorIndex,
  setSelectionAnchorIndex,
  handleSelectAll,
  handleClearSelection,
  handleDelete,
  handleFileDoubleClick
}: {
  items: ItemsResponse | undefined;
  selectedItems: Set<string>;
  setSelectedItems: React.Dispatch<React.SetStateAction<Set<string>>>;
  lastSelectedIndex: number | null;
  setLastSelectedIndex: (index: number | null) => void;
  selectionAnchorIndex: number | null;
  setSelectionAnchorIndex: (index: number | null) => void;
  handleSelectAll: () => void;
  handleClearSelection: () => void;
  handleDelete: () => void;
  handleFileDoubleClick: (fileInfo: FileInterface) => void;
}) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't interfere if user is typing in an input
      if (document.activeElement?.tagName === "INPUT" || document.activeElement?.tagName === "TEXTAREA" || document.activeElement?.hasAttribute("contenteditable")) {
        return;
      }
      
      // Don't interfere if a dialog is open
      if (document.querySelector('[role="dialog"]')) {
        return;
      }

      // We only care about specific keys
      const isArrowUp = e.key === 'ArrowUp';
      const isArrowDown = e.key === 'ArrowDown';
      const isEnter = e.key === 'Enter';
      const isDelete = e.key === 'Delete';
      const isEscape = e.key === 'Escape';
      const isA = e.key.toLowerCase() === 'a';
      
      if (!isArrowUp && !isArrowDown && !isEnter && !isDelete && !isEscape && !isA) {
        return;
      }

      // Ctrl+A / Cmd+A
      if (isA && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        handleSelectAll();
        return;
      }

      if (isEscape) {
        e.preventDefault();
        handleClearSelection();
        return;
      }

      if (isDelete && selectedItems.size > 0) {
        e.preventDefault();
        handleDelete();
        return;
      }

      if (!items?.items || items.items.length === 0) return;

      if (isEnter) {
        if (selectedItems.size === 1 && lastSelectedIndex !== null) {
          e.preventDefault();
          handleFileDoubleClick(items.items[lastSelectedIndex]);
        }
        return;
      }

      if (isArrowUp || isArrowDown) {
        e.preventDefault(); // Prevent page scrolling

        let nextIndex = 0;

        if (lastSelectedIndex !== null) {
          nextIndex = isArrowDown ? lastSelectedIndex + 1 : lastSelectedIndex - 1;
          nextIndex = Math.max(0, Math.min(nextIndex, items.items.length - 1));
        } else {
          nextIndex = isArrowDown ? 0 : items.items.length - 1;
        }

        if (e.shiftKey) {
          const anchor = selectionAnchorIndex ?? lastSelectedIndex ?? nextIndex;
          const newSet = new Set<string>();
          const start = Math.min(anchor, nextIndex);
          const end = Math.max(anchor, nextIndex);
          for (let i = start; i <= end; i++) {
            newSet.add(items.items[i].name);
          }
          setSelectedItems(newSet);
          if (selectionAnchorIndex === null) {
            setSelectionAnchorIndex(anchor);
          }
        } else {
          const newSet = new Set<string>();
          newSet.add(items.items[nextIndex].name);
          setSelectedItems(newSet);
          setSelectionAnchorIndex(nextIndex);
        }
        
        setLastSelectedIndex(nextIndex);

        // Scroll into view
        setTimeout(() => {
          const element = document.getElementById(`file-item-${nextIndex}`);
          if (element) {
            element.scrollIntoView({ block: "nearest", behavior: "smooth" });
          }
        }, 0);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => { window.removeEventListener('keydown', handleKeyDown); };
  }, [items, lastSelectedIndex, selectionAnchorIndex, selectedItems, handleSelectAll, handleClearSelection, handleDelete, handleFileDoubleClick, setSelectedItems, setSelectionAnchorIndex, setLastSelectedIndex]);
}
