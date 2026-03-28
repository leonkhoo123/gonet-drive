import { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Plus, FolderPlus, Upload, FolderUp, Loader2 } from "lucide-react";
import DefaultLayout from "@/layouts/DefaultLayout";
import VersionTag from "@/components/custom/versionTag";
import VideoPlayerModalGeneric from "@/components/custom/videoPlayerModalGeneric";
import PhotoViewerModal from "@/components/custom/photoViewerModal";
import TextViewerModal from "@/components/custom/textViewerModal";
import PdfViewerModal from "@/components/custom/pdfViewerModal";
import { useFileManager } from "@/hooks/useFileManager";
import ShareBreadcrumb from "@/components/share/ShareBreadcrumb";
import ShareToolbar from "@/components/share/ShareToolbar";
import ShareMobileSelectionToolbar from "@/components/share/ShareMobileSelectionToolbar";
import ShareFileList from "@/components/share/ShareFileList";
import SharePropertiesModal from "@/components/share/SharePropertiesModal";
import ShareDownloadDirDialog from "@/components/share/ShareDownloadDirDialog";
import ShareDeleteDialog from "@/components/share/ShareDeleteDialog";
import ShareCreateFolderDialog from "@/components/share/ShareCreateFolderDialog";
import ShareDuplicateCheckDialog from "@/components/share/ShareDuplicateCheckDialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import { setShareMode } from "@/api/api-file";
import { MusicPlayerV2 } from "@/components/custom/musicPlayerV2";
import { useAppHealth } from "@/hooks/useAppHealth";
import { OperationQueueProgress } from "@/components/custom/operationQueueProgress";
import { MobileClipboardToast } from "@/components/home/MobileClipboardToast";
import HomeRenameDialog from "@/components/home/HomeRenameDialog";
import { useTheme } from "@/components/theme-provider";

import { ShareModeToggle } from "@/components/share/ShareModeToggle";

export default function ShareHomePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [authority, setAuthority] = useState<string | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const handleAppRefresh = useCallback(() => {
    setRefreshTrigger(prev => prev + 1);
  }, []);

  const { healthData, isWsConnected, isHealthConnected } = useAppHealth(handleAppRefresh);

  const { setTheme } = useTheme();

  useEffect(() => {
    // Set global share mode for API calls
    setShareMode(true);
    
    if (!sessionStorage.getItem("share_theme_toggled")) {
      setTheme("system");
    }
    
    // Check if we have authority stored, if not redirect back to verify
    const auth = sessionStorage.getItem("share_authority");
    if (!auth || !id) {
      void navigate(id ? `/share/${id}` : '/', { replace: true });
    } else {
      setAuthority(auth);
    }

    return () => {
      setShareMode(false);
    };
  }, [id, navigate, setTheme]);

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
    shareRoot,
    selectedItems,
    handleClearSelection,
    handleSelectAll,
    handleDelete,
    confirmDelete,
    isDeleteDialogOpen,
    setIsDeleteDialogOpen,
    itemsToDelete,
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
    sortField,
    sortOrder,
    handleSortChange,
    handleCut,
    handleCopy,
    handlePaste,
    handleRename,
    clipboardItems,
    handleClearClipboard,
    isRenameDialogOpen,
    setIsRenameDialogOpen,
    confirmRename,
    itemToRename,
  } = useFileManager({ baseRoute: `/share/${id}/home`, uploadChunkSize: healthData?.upload_chunk_size });

  // Hook up the refresh trigger to actually call handleRefresh
  useEffect(() => {
    if (refreshTrigger > 0) {
      void handleRefresh();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshTrigger]);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (authority !== 'modify') return;
    if (e.target.files && e.target.files.length > 0) {
      void handleUploadFiles(Array.from(e.target.files), currentPath);
    }
    if (fileInputRef.current) fileInputRef.current.value = "";
    if (folderInputRef.current) folderInputRef.current.value = "";
  };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable
      ) {
        return;
      }

      if (e.key === "F5" || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "r")) {
        e.preventDefault();
        void handleRefresh();
        return;
      }

      if (
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
    handleBack,
    isCreateFolderDialogOpen,
    isDeleteDialogOpen,
    isPropertiesDialogOpen,
    isDownloadDirDialogOpen,
    selectedVideo,
    selectedPhoto,
  ]);

  if (!authority) {
    return (
      <DefaultLayout>
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-10 w-10 animate-spin text-muted-foreground" />
        </div>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout>
      <div className="flex flex-1 w-full overflow-hidden relative">
        <div className="flex-1 flex flex-col min-w-0 bg-background overflow-hidden" onClick={(e) => {
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
          {/* Top Title Bar */}
          <div className="h-14 border-b flex items-center justify-between px-4 bg-background shrink-0 z-10">
            <div className="flex items-center gap-2">
              <img src="/images/logo-removebg-preview.png" alt="Logo" className="w-8 h-8 object-contain" />
              <h1 className="text-lg font-bold text-foreground tracking-tight">{healthData?.service_name ?? "Shared Drive"}</h1>
            </div>
            <div className="flex items-center gap-2">
              <div className="flex flex-col gap-1 items-end">
                <div className="flex items-center gap-1.5" title={`API: ${isHealthConnected ? 'OK' : 'Error'}`}>
                  <span className="text-[10px] text-muted-foreground/80 font-mono tracking-wider leading-none">/health</span>
                  <div className={`w-1.5 h-1.5 rounded-full ${isHealthConnected ? 'bg-green-500 shadow-[0_0_4px_#22c55e]' : 'bg-red-500 shadow-[0_0_4px_#ef4444]'}`} />
                </div>
                {authority === 'modify' && (
                  <div className="flex items-center gap-1.5" title={`WS: ${isWsConnected ? 'Connected' : 'Disconnected'}`}>
                    <span className="text-[10px] text-muted-foreground/80 font-mono tracking-wider leading-none">WebSocket</span>
                    <div className={`w-1.5 h-1.5 rounded-full ${isWsConnected ? 'bg-green-500 shadow-[0_0_4px_#22c55e]' : 'bg-red-500 shadow-[0_0_4px_#ef4444]'}`} />
                  </div>
                )}
              </div>
              <div className="w-px h-6 bg-border mx-1 block"></div>
              <ShareModeToggle />
            </div>
          </div>

          <div className="md:hidden">
            {selectedItems.size > 0 ? (
              <ShareMobileSelectionToolbar
                selectedItemsSize={selectedItems.size}
                onCancel={handleClearSelection}
                onSelectAll={handleSelectAll}
                onCut={() => undefined}
                onCopy={() => undefined}
                onDelete={handleDelete}
                onProperties={() => { void handleProperties(); }}
                onDownload={handleDownload}
                authority={authority}
              />
            ) : (
              <ShareBreadcrumb 
                currentPath={currentPath} 
                shareRoot={shareRoot}
                isFolderEmpty={!items?.items || items.items.length === 0}
                isSingleFile={items?.is_single_file}
                onProperties={(name?: string, isCurrentDir?: boolean) => { void handleProperties(name, isCurrentDir); }}
                onRefresh={() => { void handleRefresh(); }}
                onDownload={() => { handleDownload(); }}
              />
            )}
          </div>

          <div className="hidden md:block">
            <ShareBreadcrumb 
              currentPath={currentPath} 
              shareRoot={shareRoot}
              isSingleFile={items?.is_single_file}
              onProperties={(name?: string, isCurrentDir?: boolean) => { void handleProperties(name, isCurrentDir); }}
              onRefresh={() => { void handleRefresh(); }}
            />
          </div>

          <div className="hidden md:block">
            <ShareToolbar
              currentPath={currentPath}
              shareRoot={shareRoot}
              selectedItemsSize={selectedItems.size}
              clipboardItemsCount={clipboardItems.items.length}
              clipboardOperation={clipboardItems.operation}
              isFolderEmpty={!items?.items || items.items.length === 0}
              isSingleFile={items?.is_single_file}
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
              onUploadFiles={(files: File[]) => { void handleUploadFiles(files, currentPath); }}
              authority={authority}
            />
          </div>

          <ShareFileList
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
            onProperties={(name?: string, isCurrentDir?: boolean) => { void handleProperties(name, isCurrentDir); }}
            onDownload={(name?: string) => { handleDownload(name); }}
            onCreateFolder={handleCreateFolder}
            clipboardItems={clipboardItems.items}
            clipboardItemsCount={clipboardItems.items.length}
            clipboardOperation={clipboardItems.operation}
            currentPath={currentPath}
            onUploadDrop={(files: File[], path: string) => { void handleUploadFiles(files, path); }}
            sortField={sortField}
            sortOrder={sortOrder}
            onSortChange={handleSortChange}
            authority={authority}
          />
        </div>

        {!selectedVideo && !selectedPhoto && authority === 'modify' && (
          <div className={clipboardItems.items.length > 0 ? "hidden md:block" : ""}>
            <OperationQueueProgress />
          </div>
        )}

        {!selectedVideo && !selectedPhoto && authority === 'modify' && (
          <MobileClipboardToast
            clipboardItemsCount={clipboardItems.items.length}
            operation={clipboardItems.operation}
            sourceDir={clipboardItems.sourceDir}
            currentPath={currentPath}
            onPaste={handlePaste}
            onClearClipboard={handleClearClipboard}
          />
        )}

        {/* Mobile Floating Action Button */}
        {!selectedVideo && !selectedPhoto && authority === 'modify' && !items?.is_single_file && (
          <div className="md:hidden absolute bottom-4 right-4 z-50">
            <input 
              type="file" 
              multiple 
              className="hidden" 
              ref={fileInputRef} 
              onChange={handleFileChange} 
            />
            <input 
              type="file" 
              className="hidden" 
              ref={folderInputRef} 
              {...{ webkitdirectory: "", directory: "" } as any} 
              onChange={handleFileChange} 
            />
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <div 
                  role="button"
                  className="w-14 h-14 bg-background text-foreground shadow-lg rounded-full border flex items-center justify-center cursor-pointer hover:bg-muted transition-all duration-300"
                >
                  <Plus className="w-7 h-7" />
                </div>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" sideOffset={12}>
                <DropdownMenuItem onClick={handleCreateFolder}>
                  <FolderPlus className="mr-2 h-4 w-4" />
                  <span>Create Folder</span>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => fileInputRef.current?.click()}>
                  <Upload className="mr-2 h-4 w-4" />
                  <span>Upload File</span>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => folderInputRef.current?.click()}>
                  <FolderUp className="mr-2 h-4 w-4" />
                  <span>Upload Folder</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </div>

      {selectedVideo && (
        <VideoPlayerModalGeneric
          file={selectedVideo}
          isOpen={!!selectedVideo}
          onClose={handlePlayerClose}
        />
      )}
      
      {selectedPhoto && (
        <PhotoViewerModal
          initialFile={selectedPhoto}
          allItems={items?.items ?? []}
          isOpen={!!selectedPhoto}
          onClose={() => { setSelectedPhoto(null); }}
        />
      )}
      <VersionTag />

      <ShareDeleteDialog
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

      <ShareCreateFolderDialog
        isOpen={isCreateFolderDialogOpen}
        onOpenChange={setIsCreateFolderDialogOpen}
        onConfirm={confirmCreateFolder}
        isLoading={isLoading}
        existingNames={(items?.items ?? []).map(item => item.name)}
      />

      <SharePropertiesModal
        isOpen={isPropertiesDialogOpen}
        onOpenChange={setIsPropertiesDialogOpen}
        data={propertiesData}
      />

      <ShareDownloadDirDialog
        isOpen={isDownloadDirDialogOpen}
        onOpenChange={setIsDownloadDirDialogOpen}
        onConfirm={confirmDownloadDir}
        currentPath={currentPath}
      />

      <ShareDuplicateCheckDialog
        isOpen={isUploadDuplicateCheckDialogOpen}
        onOpenChange={setIsUploadDuplicateCheckDialogOpen}
        onConfirm={executeUpload}
        onCancel={() => { setIsUploadDuplicateCheckDialogOpen(false); setPendingUploads(null); }}
        duplicates={uploadDuplicateItems}
        isLoading={false}
        isChecking={isUploadDuplicateChecking}
      />

      {selectedMusic && (
        <MusicPlayerV2
          file={selectedMusic} 
          playlist={(items?.items ?? []).filter(item => item.type !== 'dir' && (item.name.toLowerCase().endsWith('.mp3') || item.name.toLowerCase().endsWith('.wav') || item.name.toLowerCase().endsWith('.flac') || item.name.toLowerCase().endsWith('.ogg') || item.name.toLowerCase().endsWith('.m4a') || item.name.toLowerCase().endsWith('.aac')))}
          onSelectMusic={setSelectedMusic}
          onClose={() => { setSelectedMusic(null); }} 
          forcePause={!!selectedVideo}
        />
      )}

      {selectedDocument && (
        <TextViewerModal
          file={selectedDocument}
          isOpen={!!selectedDocument}
          onClose={() => { setSelectedDocument(null); }}
        />
      )}

      {selectedPdf && (
        <PdfViewerModal
          file={selectedPdf}
          isOpen={!!selectedPdf}
          onClose={() => { setSelectedPdf(null); }}
        />
      )}
    </DefaultLayout>
  );
}
