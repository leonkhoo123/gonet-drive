import { useState, useCallback } from "react";

export function useFileDragAndDrop(
  currentPath: string,
  isRecycleBin: boolean,
  onUploadDrop: (files: File[], targetPath: string) => void
) {
  const [isDragging, setIsDragging] = useState(false);

  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!isRecycleBin) setIsDragging(true);
  }, [isRecycleBin]);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!isRecycleBin) setIsDragging(false);
  }, [isRecycleBin]);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!isDragging && !isRecycleBin) {
      setIsDragging(true);
    }
  }, [isDragging, isRecycleBin]);

  const traverseFileTree = useCallback(async (item: FileSystemEntry, path: string, files: File[]): Promise<void> => {
    return new Promise((resolve) => {
      if (item.isFile) {
        (item as FileSystemFileEntry).file((file) => {
          // Attach custom path for folder structures on drop
          if (path) {
            Object.defineProperty(file, 'customPath', {
              value: path + file.name,
              writable: false,
            });
          }
          files.push(file);
          resolve();
        });
      } else if (item.isDirectory) {
        const dirReader = (item as FileSystemDirectoryEntry).createReader();
        dirReader.readEntries((entries: FileSystemEntry[]) => {
          void (async () => {
            for (const entry of entries) {
              await traverseFileTree(entry, path + item.name + "/", files);
            }
            resolve();
          })();
        });
      } else {
        resolve();
      }
    });
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    void (async () => {
      const items = e.dataTransfer.items;
      const files: File[] = [];

      if (items.length > 0) {
        const entries: FileSystemEntry[] = [];
        for (const item of Array.from(items)) {
          const entry = item.webkitGetAsEntry();
          if (entry) {
            entries.push(entry);
          }
        }
        for (const entry of entries) {
          await traverseFileTree(entry, "", files);
        }
      } else {
        // Fallback for older browsers
        for (const file of Array.from(e.dataTransfer.files)) {
          files.push(file);
        }
      }

      if (files.length > 0 && !isRecycleBin) {
        onUploadDrop(files, currentPath);
      }
    })();
  }, [currentPath, onUploadDrop, isRecycleBin, traverseFileTree]);

  return {
    isDragging,
    handleDragEnter,
    handleDragLeave,
    handleDragOver,
    handleDrop,
  };
}
