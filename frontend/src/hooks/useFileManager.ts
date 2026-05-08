import { useNavigate } from "react-router-dom";
import { useState } from "react";
import { encodePathToUrl } from "@/utils/utils";
import { type FileInterface } from "@/api/api-file";

import { useFileSystem } from "./useFileManager/useFileSystem";
import { useFileSelection } from "./useFileManager/useFileSelection";
import { useFileOperations } from "./useFileManager/useFileOperations";
import { useFileUpload } from "./useFileManager/useFileUpload";
import { useVideoOperations } from "./useFileManager/useVideoOperations";
import { useKeyboardShortcuts } from "./useFileManager/useKeyboardShortcuts";

export function useFileManager({ uploadChunkSize, baseRoute = "/home" }: { uploadChunkSize?: number, baseRoute?: string } = {}) {
  const navigate = useNavigate();

  // 1. Core File System (fetching, path, loading, errors)
  const {
    items,
    isLoading,
    setIsLoading,
    error,
    setError,
    currentPath,
    shareRoot,
    handleRefresh,
    sortField,
    setSortField,
    sortOrder,
    setSortOrder,
    handleSortChange
  } = useFileSystem(baseRoute);

  // 2. Video Operations (setSelectedVideo, player close, clean up)
  const {
    selectedVideo,
    setSelectedVideo,
    handlePlayerClose
  } = useVideoOperations({ currentPath, handleRefresh, setIsLoading, setError });

  const [selectedPhoto, setSelectedPhoto] = useState<FileInterface | null>(null);
  const [selectedMusic, setSelectedMusic] = useState<FileInterface | null>(null);
  const [selectedDocument, setSelectedDocument] = useState<FileInterface | null>(null);
  const [selectedPdf, setSelectedPdf] = useState<FileInterface | null>(null);

  // 3. Selection state and click handlers
  const {
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
  } = useFileSelection(items, currentPath, setSelectedVideo, setSelectedPhoto, setSelectedMusic, setSelectedDocument, setSelectedPdf, baseRoute);

  // 4. File Actions (copy, cut, paste, delete, rename, properties, folder)
  const {
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
  } = useFileOperations({ currentPath, selectedItems, setSelectedItems, handleRefresh, setIsLoading, isSingleFile: items?.is_single_file });

  // 5. Uploading files
  const { 
    handleUploadFiles,
    executeUpload,
    isUploadDuplicateCheckDialogOpen,
    setIsUploadDuplicateCheckDialogOpen,
    isUploadDuplicateChecking,
    uploadDuplicateItems,
    setPendingUploads
  } = useFileUpload(handleRefresh, uploadChunkSize);

  // 6. Keyboard Shortcuts
  useKeyboardShortcuts({
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
  });

  // Additional navigation handler
  const handleBack = () => {
    if (currentPath === "/" || currentPath === shareRoot) return;
    const pathParts = currentPath.split("/").filter(Boolean);
    pathParts.pop();
    const parentPath = pathParts.length > 0 ? "/" + pathParts.join("/") : "/";
    void navigate(baseRoute + encodePathToUrl(parentPath) + window.location.search);
  };

  return {
    items,
    isLoading,
    error,
    selectedVideo,
    setSelectedVideo,
    selectedPhoto,
    setSelectedPhoto,
    selectedMusic,
    setSelectedMusic,
    selectedDocument,
    setSelectedDocument,
    selectedPdf,
    setSelectedPdf,
    currentPath,
    shareRoot,
    selectedItems,
    clipboardItems,
    handleClearSelection,
    handleSelectAll,
    handleCut,
    handleCopy,
    handlePaste,
    handleClearClipboard,
    handleDelete,
    handleEmptyRecycleBin,
    confirmDelete,
    isDeleteDialogOpen,
    setIsDeleteDialogOpen,
    itemsToDelete,
    handleRename,
    confirmRename,
    itemToRename,
    isRenameDialogOpen,
    setIsRenameDialogOpen,
    handleCreateFolder,
    confirmCreateFolder,
    isCreateFolderDialogOpen,
    setIsCreateFolderDialogOpen,
    handleBack,
    handleFileClick,
    handleFileContextMenu,
    handleFileDoubleClick,
    handlePlayerClose,
    handleRefresh,
    handleProperties,
    propertiesData,
    isPropertiesDialogOpen,
    setIsPropertiesDialogOpen,
    isPropertiesLoading,
    isShareDialogOpen,
    setIsShareDialogOpen,
    itemToShare,
    handleShare,
    handleUploadFiles,
    executeUpload,
    isUploadDuplicateCheckDialogOpen,
    setIsUploadDuplicateCheckDialogOpen,
    isUploadDuplicateChecking,
    uploadDuplicateItems,
    setPendingUploads,
    handleDownload,
    isDownloadDirDialogOpen,
    setIsDownloadDirDialogOpen,
    confirmDownloadDir,
    isDuplicateCheckDialogOpen,
    setIsDuplicateCheckDialogOpen,
    isDuplicateChecking,
    duplicateItems,
    executePaste,
    sortField,
    setSortField,
    sortOrder,
    setSortOrder,
    handleSortChange,
  };
}
