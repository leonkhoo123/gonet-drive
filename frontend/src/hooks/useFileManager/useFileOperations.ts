import { useState } from 'react';
import { toast } from "sonner";
import { 
  copyFiles, moveFiles, deleteFiles, deletePermanentFiles, 
  renameFile, createFolder, getFileProperties, downloadFiles,
  checkDuplicates,
  type DuplicateItem,
  type PropertiesResponse 
} from "@/api/api-file";

export function useFileOperations({
  currentPath,
  selectedItems,
  setSelectedItems,
  handleRefresh,
  setIsLoading,
  isSingleFile
}: {
  currentPath: string;
  selectedItems: Set<string>;
  setSelectedItems: (items: Set<string>) => void;
  handleRefresh: () => Promise<void>;
  setIsLoading: (loading: boolean) => void;
  isSingleFile?: boolean;
}) {
  const [clipboardItems, setClipboardItems] = useState<{ items: string[], operation: 'cut' | 'copy' | null, sourceDir?: string }>({ items: [], operation: null });
  const [itemsToDelete, setItemsToDelete] = useState<Set<string> | null>(null);
  const [itemToRename, setItemToRename] = useState<string | null>(null);
  const [isCreateFolderDialogOpen, setIsCreateFolderDialogOpen] = useState(false);
  const [propertiesData, setPropertiesData] = useState<PropertiesResponse | null>(null);
  const [isPropertiesDialogOpen, setIsPropertiesDialogOpen] = useState(false);
  const [isPropertiesLoading, setIsPropertiesLoading] = useState(false);

  const [isShareDialogOpen, setIsShareDialogOpen] = useState(false);
  const [itemToShare, setItemToShare] = useState<string | null>(null);
  const [isDownloadDirDialogOpen, setIsDownloadDirDialogOpen] = useState(false);
  const [isDuplicateCheckDialogOpen, setIsDuplicateCheckDialogOpen] = useState(false);
  const [isDuplicateChecking, setIsDuplicateChecking] = useState(false);
  const [duplicateItems, setDuplicateItems] = useState<DuplicateItem[]>([]);

  const isDeleteDialogOpen = itemsToDelete !== null;
  const setIsDeleteDialogOpen = (open: boolean) => {
    if (!open) setItemsToDelete(null);
  };

  const isRenameDialogOpen = itemToRename !== null;
  const setIsRenameDialogOpen = (open: boolean) => {
    if (!open) setItemToRename(null);
  };

  const handleCut = () => {
    if (selectedItems.size === 0) {
      toast.error("No items selected");
      return;
    }
    const sources = Array.from(selectedItems).map(name =>
      currentPath === "/" ? `/${name}` : `${currentPath}/${name}`
    );
    setClipboardItems({ items: sources, operation: 'cut', sourceDir: currentPath });

    if (window.innerWidth >= 768) {
      toast.success(`${selectedItems.size} item(s) ready to move`);
    } else {
      setSelectedItems(new Set());
    }
  };

  const handleCopy = () => {
    if (selectedItems.size === 0) {
      toast.error("No items selected");
      return;
    }
    const sources = Array.from(selectedItems).map(name =>
      currentPath === "/" ? `/${name}` : `${currentPath}/${name}`
    );
    setClipboardItems({ items: sources, operation: 'copy', sourceDir: currentPath });

    if (window.innerWidth >= 768) {
      toast.success(`${selectedItems.size} item(s) added to clipboard`);
    } else {
      setSelectedItems(new Set());
    }
  };

  const executePaste = async () => {
    if (clipboardItems.items.length === 0 || !clipboardItems.operation) {
      return;
    }

    try {
      setIsLoading(true);
      setIsDuplicateCheckDialogOpen(false);
      
      if (clipboardItems.operation === 'copy') {
        await copyFiles(clipboardItems.items, currentPath);
      } else {
        await moveFiles(clipboardItems.items, currentPath);
      }
      setClipboardItems({ items: [], operation: null }); // Clear clipboard after successful paste
      await handleRefresh();
    } catch (error) {
      console.error("Paste failed:", error);
      toast.error("Paste operation failed");
    } finally {
      setIsLoading(false);
    }
  };

  const handlePaste = async () => {
    if (clipboardItems.items.length === 0 || !clipboardItems.operation) {
      toast.error("Clipboard is empty");
      return;
    }

    try {
      setIsDuplicateCheckDialogOpen(true);
      setIsDuplicateChecking(true);
      setDuplicateItems([]);
      
      const startTime = Date.now();
      const res = await checkDuplicates(clipboardItems.items, currentPath);
      
      const elapsedTime = Date.now() - startTime;
      if (elapsedTime < 300) {
        await new Promise(resolve => setTimeout(resolve, 300 - elapsedTime));
      }

      if (res.hasDuplicates) {
        setIsDuplicateChecking(false);
        setDuplicateItems(res.duplicates);
      } else {
        setIsDuplicateCheckDialogOpen(false);
        await executePaste();
        setTimeout(() => { setIsDuplicateChecking(false); }, 300);
      }
    } catch (error) {
      console.error("Check duplicates failed:", error);
      toast.error("Failed to check for duplicates");
      setIsDuplicateChecking(false);
      setIsDuplicateCheckDialogOpen(false);
    }
  };

  const handleClearClipboard = () => {
    setClipboardItems({ items: [], operation: null, sourceDir: undefined });
  };

  const handleDelete = (fileName?: string | React.MouseEvent | Event) => {
    if (fileName && typeof fileName === 'string') {
      setItemsToDelete(new Set([fileName]));
      return;
    }
    if (selectedItems.size === 0) return;
    setItemsToDelete(new Set(selectedItems));
  };

  const handleEmptyRecycleBin = (allItemNames: string[]) => {
    if (allItemNames.length > 0) {
      setItemsToDelete(new Set(allItemNames));
    } else {
      toast.info("Recycle bin is already empty");
    }
  };

  const confirmDelete = async () => {
    if (!itemsToDelete || itemsToDelete.size === 0) return;
    try {
      setItemsToDelete(null); // close modal immediately
      setIsLoading(true);
      const sources = Array.from(itemsToDelete).map(name =>
        currentPath === "/" ? `/${name}` : `${currentPath}/${name}`
      );
      
      if (currentPath.startsWith("/.cloud_delete") || currentPath === "/.cloud_delete") {
        await deletePermanentFiles(sources);
      } else {
        await deleteFiles(sources);
      }

      setSelectedItems(new Set());
      await handleRefresh();
    } catch (error) {
      console.error("Delete failed:", error);
      toast.error("Delete operation failed");
    } finally {
      setIsLoading(false);
    }
  };

  const handleRename = (fileName?: string | React.MouseEvent | Event) => {
    if (fileName && typeof fileName === 'string') {
      setItemToRename(fileName);
      return;
    }
    if (selectedItems.size !== 1) {
      toast.error("Please select exactly one item to rename");
      return;
    }
    const oldName = Array.from(selectedItems)[0];
    setItemToRename(oldName);
  };

  const confirmRename = async (newName: string) => {
    if (!itemToRename || !newName || newName === itemToRename) {
      setItemToRename(null);
      return;
    }

    try {
      const oldName = itemToRename;
      setItemToRename(null);
      setIsLoading(true);
      const source = currentPath === "/" ? `/${oldName}` : `${currentPath}/${oldName}`;
      await renameFile(source, newName);
      setSelectedItems(new Set());
      await handleRefresh();
    } catch (error) {
      console.error("Rename failed:", error);
      toast.error("Rename operation failed");
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateFolder = () => {
    setIsCreateFolderDialogOpen(true);
  };

  const confirmCreateFolder = async (folderName: string) => {
    if (!folderName) {
      setIsCreateFolderDialogOpen(false);
      return;
    }

    try {
      setIsCreateFolderDialogOpen(false);
      setIsLoading(true);
      await createFolder(currentPath, folderName);
      await handleRefresh();
    } catch (error) {
      console.error("Create folder failed:", error);
      toast.error("Create folder failed");
    } finally {
      setIsLoading(false);
    }
  };

  const handleDownload = (fileName?: string | React.MouseEvent | Event) => {
    const targetFileName = typeof fileName === 'string' ? fileName : undefined;
    if (!targetFileName && selectedItems.size === 0) {
      setIsDownloadDirDialogOpen(true);
      return;
    }

    let sources: string[];
    if (isSingleFile) {
      sources = ["/"];
    } else if (targetFileName) {
      sources = [currentPath === "/" ? `/${targetFileName}` : `${currentPath}/${targetFileName}`];
    } else {
      sources = Array.from(selectedItems).map(name =>
        currentPath === "/" ? `/${name}` : `${currentPath}/${name}`
      );
    }
    
    downloadFiles(sources);
  };

  const confirmDownloadDir = () => {
    downloadFiles([currentPath]);
    setIsDownloadDirDialogOpen(false);
  };

  const handleShare = (fileName?: string | React.MouseEvent | Event, isCurrentDir = false) => {
    const targetFileName = typeof fileName === 'string' ? fileName : undefined;
    if (!isCurrentDir && !targetFileName && selectedItems.size === 0) return;

    let sharePath = '';
    if (isCurrentDir) {
      sharePath = currentPath;
    } else {
      const itemToFetch = targetFileName ?? Array.from(selectedItems)[0];
      if (!itemToFetch) return;
      sharePath = currentPath === "/" ? `/${itemToFetch}` : `${currentPath}/${itemToFetch}`;
    }
    
    setItemToShare(sharePath);
    setIsShareDialogOpen(true);
  };

  const handleProperties = async (fileName?: string | React.MouseEvent | Event, isCurrentDir = false) => {
    const targetFileName = typeof fileName === 'string' ? fileName : undefined;
    if (!isCurrentDir && !targetFileName && selectedItems.size === 0) return;
    
    setPropertiesData(null);
    setIsPropertiesDialogOpen(true);
    setIsPropertiesLoading(true);

    try {
      let sources: string[];
      if (isSingleFile) {
        sources = ["/"];
      } else if (isCurrentDir) {
        sources = [currentPath];
      } else {
        const itemsToFetch = targetFileName ? [targetFileName] : Array.from(selectedItems);
        sources = itemsToFetch.map(name =>
          currentPath === "/" ? `/${name}` : `${currentPath}/${name}`
        );
      }
      const data = await getFileProperties(sources);
      setPropertiesData(data);
    } catch (error) {
      console.error("Get properties failed:", error);
      toast.error("Failed to get properties");
      setIsPropertiesDialogOpen(false);
    } finally {
      setIsPropertiesLoading(false);
    }
  };

  return {
    clipboardItems,
    handleCut,
    handleCopy,
    handlePaste,
    handleClearClipboard,
    itemsToDelete,
    isDeleteDialogOpen,
    setIsDeleteDialogOpen,
    handleDelete,
    handleEmptyRecycleBin,
    confirmDelete,
    itemToRename,
    isRenameDialogOpen,
    setIsRenameDialogOpen,
    handleRename,
    confirmRename,
    isCreateFolderDialogOpen,
    setIsCreateFolderDialogOpen,
    handleCreateFolder,
    confirmCreateFolder,
    propertiesData,
    isPropertiesDialogOpen,
    setIsPropertiesDialogOpen,
    isPropertiesLoading,
    handleProperties,
    isShareDialogOpen,
    setIsShareDialogOpen,
    itemToShare,
    handleShare,
    handleDownload,
    isDownloadDirDialogOpen,
    setIsDownloadDirDialogOpen,
    confirmDownloadDir,
    isDuplicateCheckDialogOpen,
    setIsDuplicateCheckDialogOpen,
    isDuplicateChecking,
    duplicateItems,
    executePaste
  };
}
