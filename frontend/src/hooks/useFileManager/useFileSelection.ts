import { useState, useEffect } from 'react';
import { useLocation, useNavigate } from "react-router-dom";
import { type ItemsResponse, type FileInterface } from "@/api/api-file";
import { encodePathToUrl} from "@/utils/utils";

export function useFileSelection(
  items: ItemsResponse | undefined, 
  currentPath: string,
  setSelectedVideo: (video: FileInterface | null) => void,
  setSelectedPhoto: (photo: FileInterface | null) => void,
  setSelectedMusic: (music: FileInterface | null) => void,
  setSelectedDocument: (document: FileInterface | null) => void,
  setSelectedPdf: (pdf: FileInterface | null) => void,
  baseRoute = "/home"
) {
  const location = useLocation();
  const navigate = useNavigate();
  const [selectedItems, setSelectedItems] = useState<Set<string>>(() => new Set());
  const [lastSelectedIndex, setLastSelectedIndex] = useState<number | null>(null);
  const [selectionAnchorIndex, setSelectionAnchorIndex] = useState<number | null>(null);

  // Clear selections when changing directories
  useEffect(() => {
    setSelectedItems(new Set());
    setLastSelectedIndex(null);
    setSelectionAnchorIndex(null);
  }, [location]);

  const handleClearSelection = () => {
    setSelectedItems(new Set());
    setLastSelectedIndex(null);
    setSelectionAnchorIndex(null);
  };

  const handleSelectAll = () => {
    if (!items?.items) return;
    const allNames = new Set(items.items.map(item => item.name));
    setSelectedItems(allNames);
  };

  const handleFileClick = (fileInfo: FileInterface, index: number, event: React.MouseEvent) => {
    event.stopPropagation();
    
    // On touch devices (pointer: coarse), a single click/tap should enter the folder or open the file
    // unless they are using modifier keys (which is rare on touch, but still handled).
    const isTouchDevice = window.matchMedia("(pointer: coarse)").matches;
    
    if (isTouchDevice && !event.shiftKey && !event.ctrlKey && !event.metaKey) {
      if (selectedItems.size > 0) {
        setSelectedItems(prev => {
          const newSet = new Set(prev);
          if (newSet.has(fileInfo.name)) {
            newSet.delete(fileInfo.name);
          } else {
            newSet.add(fileInfo.name);
          }
          setSelectionAnchorIndex(index);
          setLastSelectedIndex(index);
          return newSet;
        });
        return;
      }

      if (fileInfo.media_type === "video") {
        setSelectedVideo(fileInfo);
      } else if (fileInfo.media_type === "music") {
        setSelectedMusic(fileInfo);
      } else if (fileInfo.media_type === "photo") {
        setSelectedPhoto(fileInfo);
      } else if (fileInfo.media_type === "text_documents") {
        setSelectedDocument(fileInfo);
      } else if (fileInfo.media_type === "pdf") {
        setSelectedPdf(fileInfo);
      } else if (fileInfo.type === "dir") {
        const newPath = currentPath === "/" ? `/${fileInfo.name}` : `${currentPath}/${fileInfo.name}`;
        void navigate(baseRoute + encodePathToUrl(newPath) + window.location.search);
      }
      return;
    }

    setSelectedItems(prev => {
      let newSet = new Set(prev);
      
      if (event.shiftKey && selectionAnchorIndex !== null && items?.items) {
        // Shift+Click: Select range and clear others
        newSet = new Set();
        const start = Math.min(selectionAnchorIndex, index);
        const end = Math.max(selectionAnchorIndex, index);
        for (let i = start; i <= end; i++) {
          newSet.add(items.items[i].name);
        }
      } else if (event.ctrlKey || event.metaKey) {
        // Ctrl+Click: Toggle selection
        if (newSet.has(fileInfo.name)) {
          newSet.delete(fileInfo.name);
        } else {
          newSet.add(fileInfo.name);
        }
        setSelectionAnchorIndex(index);
      } else {
        // Normal click: Single selection
        newSet = new Set();
        newSet.add(fileInfo.name);
        setSelectionAnchorIndex(index);
      }
      
      setLastSelectedIndex(index);
      return newSet;
    });
  };

  const handleFileContextMenu = (fileInfo: FileInterface, index: number) => {
    setSelectedItems(prev => {
      if (prev.has(fileInfo.name)) {
        return prev;
      }
      const newSet = new Set<string>();
      newSet.add(fileInfo.name);
      setSelectionAnchorIndex(index);
      setLastSelectedIndex(index);
      return newSet;
    });
  };

  const handleFileDoubleClick = (fileInfo: FileInterface) => {
    const isTouchDevice = window.matchMedia("(pointer: coarse)").matches;
    if (isTouchDevice) {
      // Prevent double click on touch devices to avoid triggering twice if double tap happens
      return;
    }

    if (fileInfo.media_type === "video") {
      setSelectedVideo({ ...fileInfo });
    } else if (fileInfo.media_type === "photo") {
      setSelectedPhoto({ ...fileInfo });
    }else if (fileInfo.media_type === "music") {
      setSelectedMusic({ ...fileInfo });
    } else if (fileInfo.media_type === "text_documents") {
      setSelectedDocument({ ...fileInfo });
    } else if (fileInfo.media_type === "pdf") {
      setSelectedPdf({ ...fileInfo });
    } else if (fileInfo.type === "dir") {
      const newPath = currentPath === "/" ? `/${fileInfo.name}` : `${currentPath}/${fileInfo.name}`;
      void navigate(baseRoute + encodePathToUrl(newPath) + window.location.search);
    }
  };

  return {
    selectedItems,
    setSelectedItems,
    lastSelectedIndex,
    setLastSelectedIndex,
    selectionAnchorIndex,
    setSelectionAnchorIndex,
    handleClearSelection,
    handleSelectAll,
    handleFileClick,
    handleFileContextMenu,
    handleFileDoubleClick
  };
}
