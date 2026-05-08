import { useState, useEffect, useCallback, lazy, Suspense } from "react";
import DefaultLayout from "@/layouts/DefaultLayout";
import VersionTag from "@/components/custom/versionTag";

// Lazy load heavy viewer components
const VideoPlayerModalGeneric = lazy(() => import("@/components/custom/videoPlayerModalGeneric"));
const VideoPlayerModalV2 = lazy(() => import("@/components/custom/videoPlayerModalV2"));
const PhotoViewerModal = lazy(() => import("@/components/custom/photoViewerModal"));
const TextViewerModal = lazy(() => import("@/components/custom/textViewerModal"));
const PdfViewerModal = lazy(() => import("@/components/custom/pdfViewerModal"));
const MusicPreviewModal = lazy(() => import("@/components/custom/musicPreviewModal").then(module => ({ default: module.MusicPreviewModal })));

import { useFileManager } from "@/hooks/useFileManager";
import HomeSidebar from "@/components/home/HomeSidebar";
import HomeBreadcrumb from "@/components/home/HomeBreadcrumb";
import HomeToolbar from "@/components/home/HomeToolbar";
import MobileSelectionToolbar from "@/components/home/MobileSelectionToolbar";
import HomeFileList from "@/components/home/HomeFileList";
import { OperationQueueProgress } from "@/components/custom/operationQueueProgress";
import HomePropertiesModal from "@/components/home/HomePropertiesModal";
import HomeShareDialog from "@/components/home/HomeShareDialog";
import HomeDownloadDirDialog from "@/components/home/HomeDownloadDirDialog";
import HomeDeleteDialog from "@/components/home/HomeDeleteDialog";
import HomeRenameDialog from "@/components/home/HomeRenameDialog";
import HomeCreateFolderDialog from "@/components/home/HomeCreateFolderDialog";
import HomeDuplicateCheckDialog from "@/components/home/HomeDuplicateCheckDialog";
import { useAppHealth } from "@/hooks/useAppHealth";
import { useHomeKeyboardShortcuts } from "@/hooks/useHomeKeyboardShortcuts";
import { MobileClipboardToast } from "@/components/home/MobileClipboardToast";
import { MobileFloatingActionButton } from "@/components/home/MobileFloatingActionButton";

export default function HomePage() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(window.innerWidth >= 1024);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // We need a stable handleRefresh for useAppHealth, but useFileManager needs healthData
  // We'll use a refresh trigger trick or just pass a wrapper
  const handleAppRefresh = useCallback(() => {
    setRefreshTrigger(prev => prev + 1);
  }, []);

  const { healthData, isWsConnected, isHealthConnected } = useAppHealth(handleAppRefresh);
  
  const {
    items,
    isLoading,
    error,
    selectedVideo,
    selectedPhoto,
    setSelectedPhoto,
    selectedMusic,
    setSelectedMusic,
    selectedDocument,
    setSelectedDocument,
    selectedPdf,
    setSelectedPdf,
    currentPath,
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
  } = useFileManager({ uploadChunkSize: healthData?.upload_chunk_size });

  // Hook up the refresh trigger to actually call handleRefresh
  useEffect(() => {
    if (refreshTrigger > 0) {
      void handleRefresh();
    }
  }, [refreshTrigger, handleRefresh]);

  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 1024) {
        setIsSidebarOpen(false);
      } else {
        setIsSidebarOpen(true);
      }
    };
    
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); };
  }, []);

  useHomeKeyboardShortcuts({
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
  });

  const toggleSidebar = () => { setIsSidebarOpen(!isSidebarOpen); };

  return (
    <DefaultLayout>
      <div className="flex flex-1 w-full overflow-hidden relative">
        <HomeSidebar 
          isOpen={isSidebarOpen} 
          onClose={() => { setIsSidebarOpen(false); }} 
          isWsConnected={isWsConnected}
          isHealthConnected={isHealthConnected}
          titleName={healthData?.service_name}
          storageUsage={items?.storage}
        />

        <div className="flex-1 flex flex-col min-w-0 bg-background overflow-hidden" onClick={(e) => {
          // Prevent clearing selection if clicking inside a toolbar button, menu, dialog, or popover
          const target = e.target as HTMLElement;
          if (
            target.closest('button') ||
            target.closest('[role="menuitem"]') ||
            target.closest('[role="dialog"]') ||
            target.closest('[data-radix-popper-content-wrapper]')
          ) {
            return;
          }
          handleClearSelection();
        }}>
          <div className="md:hidden">
            {selectedItems.size > 0 ? (
              <MobileSelectionToolbar
                selectedItemsSize={selectedItems.size}
                onCancel={handleClearSelection}
                onSelectAll={handleSelectAll}
                onCut={handleCut}
                onCopy={handleCopy}
                onDelete={handleDelete}
                onProperties={() => { void handleProperties(); }}
                onDownload={handleDownload}
                isRecycleBin={currentPath === '/.cloud_delete' || currentPath.startsWith('/.cloud_delete/')}
                isRecycleBinSelected={selectedItems.has('.cloud_delete')}
              />
            ) : (
              <HomeBreadcrumb
                currentPath={currentPath}
                isFolderEmpty={!items?.items || items.items.length === 0}
                onToggleSidebar={toggleSidebar}
                onProperties={(name, isCurrentDir) => { void handleProperties(name, isCurrentDir); }}
                onShare={(name: string | undefined, isCurrentDir: boolean | undefined) => { handleShare(name, isCurrentDir); }}
                onRefresh={() => { void handleRefresh(); }}
                onDownload={() => { handleDownload(); }}
                onEmptyRecycleBin={() => { handleEmptyRecycleBin((items?.items ?? []).map(i => i.name)); }}
                onCreateFolder={handleCreateFolder}
              />
            )}
          </div>

          <div className="hidden md:block">
            <HomeBreadcrumb
              currentPath={currentPath}
              onToggleSidebar={toggleSidebar}
              onProperties={(name, isCurrentDir) => { void handleProperties(name, isCurrentDir); }}
              onShare={(name, isCurrentDir) => { handleShare(name, isCurrentDir); }}
              onRefresh={() => { void handleRefresh(); }}
              onEmptyRecycleBin={() => { handleEmptyRecycleBin((items?.items ?? []).map(i => i.name)); }}
              onCreateFolder={handleCreateFolder}
            />
          </div>

          <div className="hidden md:block">
            <HomeToolbar
              currentPath={currentPath}
              selectedItemsSize={selectedItems.size}
              clipboardItemsCount={clipboardItems.items.length}
              clipboardOperation={clipboardItems.operation}
              clipboardSourceDir={clipboardItems.sourceDir}
              isFolderEmpty={!items?.items || items.items.length === 0}
              onBack={handleBack}
              onSelectAll={handleSelectAll}
              onCut={handleCut}
              onCopy={handleCopy}
              onPaste={() => { void handlePaste(); }}
              onRename={handleRename}
              onDelete={handleDelete}
              onDownload={handleDownload}
              onRefresh={() => { void handleRefresh(); }}
              onCreateFolder={handleCreateFolder}
              onUploadFiles={(files) => { void handleUploadFiles(files, currentPath); }}
              isRecycleBinSelected={selectedItems.has('.cloud_delete')}
              onEmptyRecycleBin={() => { handleEmptyRecycleBin((items?.items ?? []).map(i => i.name)); }}
            />
          </div>

          <HomeFileList
            isLoading={isLoading}
            error={error}
            items={items}
            selectedItems={selectedItems}
            onRefresh={() => { void handleRefresh(); }}
            onFileClick={handleFileClick}
            onFileContextMenu={handleFileContextMenu}
            onFileDoubleClick={handleFileDoubleClick}
            onCut={handleCut}
            onCopy={handleCopy}
            onRename={handleRename}
            onDelete={handleDelete}
            onPaste={() => { void handlePaste(); }}
            onProperties={(name, isCurrentDir) => { void handleProperties(name, isCurrentDir); }}
            onShare={(name, isCurrentDir) => { handleShare(name, isCurrentDir); }}
            onDownload={(name) => { handleDownload(name); }}
            onCreateFolder={handleCreateFolder}
            clipboardItems={clipboardItems.items}
            clipboardItemsCount={clipboardItems.items.length}
            clipboardOperation={clipboardItems.operation}
            clipboardSourceDir={clipboardItems.sourceDir}
            currentPath={currentPath}
            onUploadDrop={(files, path) => { void handleUploadFiles(files, path); }}
            sortField={sortField}
            setSortField={setSortField}
            sortOrder={sortOrder}
            setSortOrder={setSortOrder}
            onSortChange={handleSortChange}
          />
        </div>

        <div className={clipboardItems.items.length > 0 ? "hidden md:block" : ""}>
          <OperationQueueProgress />
        </div>

        <MobileClipboardToast
          clipboardItemsCount={clipboardItems.items.length}
          operation={clipboardItems.operation}
          sourceDir={clipboardItems.sourceDir}
          currentPath={currentPath}
          onPaste={handlePaste}
          onClearClipboard={handleClearClipboard}
        />

        <MobileFloatingActionButton
          isHidden={clipboardItems.items.length > 0}
          onCreateFolder={handleCreateFolder}
          onUploadFiles={(files) => handleUploadFiles(files, currentPath)}
        />
      </div>

      {selectedVideo && (
        <Suspense fallback={null}>
          {healthData?.video_mode === "custom" ? (
            <VideoPlayerModalV2
              file={selectedVideo}
              isOpen={!!selectedVideo}
              onClose={handlePlayerClose}
            />
          ) : (
            <VideoPlayerModalGeneric
              file={selectedVideo}
              isOpen={!!selectedVideo}
              onClose={handlePlayerClose}
            />
          )}
        </Suspense>
      )}
      
      {selectedPhoto && (
        <Suspense fallback={null}>
          <PhotoViewerModal
            initialFile={selectedPhoto}
            allItems={items?.items ?? []}
            isOpen={!!selectedPhoto}
            onClose={() => { setSelectedPhoto(null); }}
          />
        </Suspense>
      )}
      <VersionTag />

      <HomeDeleteDialog
        isOpen={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        onConfirm={confirmDelete}
        itemsToDeleteSize={itemsToDelete?.size ?? 0}
        currentPath={currentPath}
        isLoading={isLoading}
      />

      <HomeRenameDialog
        isOpen={isRenameDialogOpen}
        onOpenChange={setIsRenameDialogOpen}
        onConfirm={confirmRename}
        itemToRename={itemToRename}
        isLoading={isLoading}
        existingNames={(items?.items ?? []).map(item => item.name)}
      />

      <HomeCreateFolderDialog
        isOpen={isCreateFolderDialogOpen}
        onOpenChange={setIsCreateFolderDialogOpen}
        onConfirm={confirmCreateFolder}
        isLoading={isLoading}
        existingNames={(items?.items ?? []).map(item => item.name)}
      />

      <HomePropertiesModal
        isOpen={isPropertiesDialogOpen}
        onOpenChange={setIsPropertiesDialogOpen}
        data={propertiesData}
      />

      <HomeShareDialog
        isOpen={isShareDialogOpen}
        onOpenChange={setIsShareDialogOpen}
        itemPath={itemToShare}
      />

      <HomeDownloadDirDialog
        isOpen={isDownloadDirDialogOpen}
        onOpenChange={setIsDownloadDirDialogOpen}
        onConfirm={confirmDownloadDir}
        currentPath={currentPath}
      />

      <HomeDuplicateCheckDialog
        isOpen={isDuplicateCheckDialogOpen}
        onOpenChange={setIsDuplicateCheckDialogOpen}
        onConfirm={executePaste}
        onCancel={() => { setIsDuplicateCheckDialogOpen(false); }}
        duplicates={duplicateItems}
        isLoading={isLoading}
        isChecking={isDuplicateChecking}
      />

      <HomeDuplicateCheckDialog
        isOpen={isUploadDuplicateCheckDialogOpen}
        onOpenChange={setIsUploadDuplicateCheckDialogOpen}
        onConfirm={executeUpload}
        onCancel={() => { setIsUploadDuplicateCheckDialogOpen(false); setPendingUploads(null); }}
        duplicates={uploadDuplicateItems}
        isLoading={false}
        isChecking={isUploadDuplicateChecking}
      />

      {selectedMusic && (
        <Suspense fallback={null}>
          <MusicPreviewModal
            file={selectedMusic} 
            isOpen={!!selectedMusic}
            onClose={() => { setSelectedMusic(null); }} 
          />
        </Suspense>
      )}

      {selectedDocument && (
        <Suspense fallback={null}>
          <TextViewerModal
            file={selectedDocument}
            isOpen={!!selectedDocument}
            onClose={() => { setSelectedDocument(null); }}
          />
        </Suspense>
      )}

      {selectedPdf && (
        <Suspense fallback={null}>
          <PdfViewerModal
            file={selectedPdf}
            isOpen={!!selectedPdf}
            onClose={() => { setSelectedPdf(null); }}
          />
        </Suspense>
      )}
    </DefaultLayout>
  );
}
