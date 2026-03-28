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

  const traverseFileTree = async (item: any, path: string, files: File[]): Promise<void> => {
    return new Promise((resolve) => {
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      if (item.isFile) {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
        item.file((file: File) => {
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
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      } else if (item.isDirectory) {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call, @typescript-eslint/no-unsafe-assignment
        const dirReader = item.createReader();
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
        dirReader.readEntries(async (entries: any[]) => {
          for (const entry of entries) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/restrict-plus-operands
            await traverseFileTree(entry, path + item.name + "/", files);
          }
          resolve();
        });
      } else {
        resolve();
      }
    });
  };

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    void (async () => {
      const items = e.dataTransfer.items;
      const files: File[] = [];

      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      if (items) {
        const entries = [];
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
  }, [currentPath, onUploadDrop, isRecycleBin]);

  return {
    isDragging,
    handleDragEnter,
    handleDragLeave,
    handleDragOver,
    handleDrop,
  };
}
