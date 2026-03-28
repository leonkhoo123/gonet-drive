import { Folder, UploadCloud, Loader2, ArrowUp, ArrowDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { ItemsResponse, FileInterface } from "@/api/api-file";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Clipboard, Info } from "lucide-react";
import { useState, useEffect, useRef } from "react";
import { useFileDragAndDrop } from "@/hooks/useFileManager/useFileDragAndDrop";
import { useFileInteraction } from "@/hooks/useFileManager/useFileInteraction";
import { ShareFileListItem } from "./ShareFileListItem";

interface ShareFileListProps {
  isLoading: boolean;
  error: boolean;
  items?: ItemsResponse;
  selectedItems: Set<string>;
  onRefresh: () => void;
  onFileClick: (fileInfo: FileInterface, index: number, event: React.MouseEvent) => void;
  onFileDoubleClick: (fileInfo: FileInterface) => void;
  onFileContextMenu: (fileInfo: FileInterface, index: number) => void;
  onCut: () => void;
  onCopy: () => void;
  onRename: (fileName?: string) => void;
  onDelete: (fileName?: string) => void;
  onPaste: () => void;
  onProperties: (fileName?: string, isCurrentDir?: boolean) => void;
  onDownload: (fileName?: string) => void;
  onCreateFolder: () => void;
  clipboardItems: string[];
  clipboardItemsCount: number;
  clipboardOperation: 'cut' | 'copy' | null; 
  clipboardSourceDir?: string;
  currentPath: string;
  onUploadDrop: (files: File[], targetPath: string) => void;
  sortField?: 'name' | 'size' | 'modified' | null;
  sortOrder?: 'asc' | 'desc';
  onSortChange?: (field: 'name' | 'size' | 'modified') => void;
  authority: string;
}

export default function ShareFileList({
  isLoading,
  error,
  items,
  selectedItems,
  onRefresh,
  onFileClick,
  onFileDoubleClick,
  onFileContextMenu,
  onCut,
  onCopy,
  onRename,
  onDelete,
  onPaste,
  onProperties,
  onDownload,
  onCreateFolder,
  clipboardItems,
  clipboardItemsCount,
  clipboardOperation,
  clipboardSourceDir,
  currentPath,
  onUploadDrop,
  sortField,
  sortOrder,
  onSortChange,
  authority,
}: ShareFileListProps) {
  const [displayItems, setDisplayItems] = useState<ItemsResponse | undefined>(items);
  const [openDropdownName, setOpenDropdownName] = useState<string | null>(null);
  const isTouchDevice = window.matchMedia("(pointer: coarse)").matches;

  const isRecycleBin = currentPath === '/.cloud_delete' || currentPath.startsWith('/.cloud_delete/');

  const {
    isDragging,
    handleDragEnter,
    handleDragLeave,
    handleDragOver,
    handleDrop,
  } = useFileDragAndDrop(currentPath, isRecycleBin || (items?.is_single_file ?? false), onUploadDrop);

  const {
    handleTouchStart,
    handleTouchEnd,
    handleTouchMove,
    handleItemClick,
    handleItemDoubleClick,
    transitioningFolder,
    setTransitioningFolder,
  } = useFileInteraction({
    onFileClick,
    onFileDoubleClick,
    onFileContextMenu,
    selectedItems,
    isTouchDevice,
  });

  useEffect(() => {
    if (items) {
      setDisplayItems(items);
      setTransitioningFolder(null); // Reset transition state when items change
    }
  }, [items, setTransitioningFolder]);

  const scrollContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTop = 0;
    }
  }, [currentPath]);

  const fileListContainer = (
    <div 
      ref={scrollContainerRef}
      className="flex-1 min-h-0 relative overflow-y-scroll overscroll-y-none p-0 md:p-3 scrollbar-thumb-rounded-full scrollbar-track-rounded-full scrollbar scrollbar-thumb-black/20 dark:scrollbar-thumb-white/20 scrollbar-track-transparent"
    >
      {/* Loading Overlay */}
      <div 
        className={`absolute inset-0 flex flex-col items-center justify-center transition-opacity duration-300 pointer-events-none z-10 ${
          isLoading && !items ? 'opacity-100' : 'opacity-0'
        }`}
      >
        <div className="flex flex-col items-center justify-center text-muted-foreground">
          <Loader2 className="h-10 w-10 animate-spin text-primary mb-4" />
          <p>Loading</p>
        </div>
      </div>

      {/* Content Layer */}
      <div 
        className={`transition-opacity duration-300 min-h-full ${
          isLoading && !items ? 'opacity-0 pointer-events-none' : 'opacity-100'
        }`}
      >
        {error ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground min-h-[50vh]">
            <p className="text-red-500 mb-2">Failed to load directory.</p>
            <Button variant="outline" size="sm" onClick={onRefresh}>Try Again</Button>
          </div>
        ) : !displayItems?.items || displayItems.items.length === 0 ? (
          <div className={`flex flex-col items-center justify-center h-full text-muted-foreground min-h-[50vh] transition-opacity duration-300 ${isLoading ? 'opacity-30 pointer-events-none' : 'opacity-60'}`}>
            <Folder className="h-16 w-16 mb-4 opacity-20" />
            <p>This folder is empty.</p>
          </div>
        ) : (
            <div className={`space-y-1 transition-opacity duration-300 ${isLoading ? 'opacity-50 pointer-events-none' : ''}`}>
              {displayItems.items.map((file, index) => {
                const isSelected = selectedItems.has(file.name);
                const isHidden = file.name.startsWith('.');
                const filePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`;
                const isCut = clipboardOperation === 'cut' && clipboardItems.includes(filePath);

                return (
                  <ShareFileListItem
                    key={file.name}
                    file={file}
                    index={index}
                    isSelected={isSelected}
                    isHidden={isHidden}
                    isCut={isCut}
                    isRecycleBin={isRecycleBin}
                    isTouchDevice={isTouchDevice}
                    transitioningFolder={transitioningFolder}
                    openDropdownName={openDropdownName}
                    setOpenDropdownName={setOpenDropdownName}
                    handleItemClick={handleItemClick}
                    handleItemDoubleClick={handleItemDoubleClick}
                    handleTouchStart={handleTouchStart}
                    handleTouchEnd={handleTouchEnd}
                    handleTouchMove={handleTouchMove}
                    onRename={() => { onRename(file.name); }}
                    onDelete={() => { onDelete(file.name); }}
                    onDownload={() => { onDownload(file.name); }}
                    onProperties={() => { onProperties(file.name); }}
                    onCut={onCut}
                    onCopy={onCopy}
                    onPaste={onPaste}
                    onFileContextMenu={onFileContextMenu}
                    authority={authority}
                    clipboardItemsCount={clipboardItemsCount}
                    clipboardOperation={clipboardOperation}
                    clipboardSourceDir={clipboardSourceDir}
                    currentPath={currentPath}
                    selectedItemsSize={selectedItems.size}
                    hasSelectedDelete={selectedItems.has('.cloud_delete')}
                  />
                );
              })}
            
            {/* Counts acting as spacer */}
            <div className="flex items-center justify-center text-sm text-muted-foreground min-h-[64px] md:min-h-[44px] border-t mt-2 border-border/50 [-webkit-touch-callout:none] [-webkit-tap-highlight-color:transparent]">
              {displayItems.folder_count !== undefined ? (
                <span>{displayItems.folder_count} folder(s), {displayItems.file_count} file(s), total {displayItems.count} item(s)</span>
              ) : (
                <span>total {displayItems.items.length} item(s)</span>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );

  return (
    <div 
      className="flex flex-col h-full w-full relative min-h-0"
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {/* Drag Overlay */}
      {isDragging && (
        <div className="absolute inset-0 z-50 bg-primary/10 backdrop-blur-sm border-2 border-dashed border-primary rounded-lg flex flex-col items-center justify-center pointer-events-none transition-all">
          <UploadCloud className="w-16 h-16 text-primary mb-4 animate-bounce" />
          <h3 className="text-2xl font-bold text-primary">Drop files here to upload</h3>
          <p className="text-primary/80 mt-2">Uploading to: {currentPath === "/" ? "Root" : currentPath}</p>
        </div>
      )}

      {/* Table Header */}
      <div className="flex border-b font-semibold py-3 md:py-2 px-6 md:pl-5 md:pr-8 text-base md:text-sm bg-muted/30 shrink-0 overflow-y-scroll scrollbar scrollbar-thumb-transparent scrollbar-track-transparent">
        <div 
          className="flex-1 text-left text-muted-foreground flex items-center cursor-pointer hover:text-foreground transition-colors group"
          onClick={() => onSortChange?.('name')}
        >
          Name
          {sortField === 'name' && (
            sortOrder === 'asc' ? <ArrowUp className="ml-1 h-3 w-3" /> : <ArrowDown className="ml-1 h-3 w-3" />
          )}
        </div>
        <div 
          className="w-24 md:w-32 hidden lg:flex justify-end text-muted-foreground items-center cursor-pointer hover:text-foreground transition-colors group"
          onClick={() => onSortChange?.('size')}
        >
          Size
          {sortField === 'size' && (
            sortOrder === 'asc' ? <ArrowUp className="ml-1 h-3 w-3" /> : <ArrowDown className="ml-1 h-3 w-3" />
          )}
        </div>
        <div 
          className="w-32 md:w-48 hidden lg:flex justify-end text-muted-foreground items-center cursor-pointer hover:text-foreground transition-colors group"
          onClick={() => onSortChange?.('modified')}
        >
          Last Modified
          {sortField === 'modified' && (
            sortOrder === 'asc' ? <ArrowUp className="ml-1 h-3 w-3" /> : <ArrowDown className="ml-1 h-3 w-3" />
          )}
        </div>
      </div>

      {isTouchDevice ? (
        fileListContainer
      ) : (
        <ContextMenu>
          <ContextMenuTrigger asChild>
            {fileListContainer}
          </ContextMenuTrigger>
          <ContextMenuContent className="w-64">
            {authority === 'modify' && (
              <ContextMenuItem onClick={(e) => { e.stopPropagation(); onCreateFolder(); }} disabled={isRecycleBin || items?.is_single_file}>
                <Folder className="mr-2 h-4 w-4" />
                New Folder
              </ContextMenuItem>
            )}
            {authority === 'modify' && (
              <ContextMenuItem 
                onClick={(e) => { e.stopPropagation(); onPaste(); }} 
                disabled={clipboardItemsCount === 0 || (clipboardOperation === 'cut' && clipboardSourceDir === currentPath) || isRecycleBin || items?.is_single_file}
              >
                <Clipboard className="mr-2 h-4 w-4" />
                Paste
              </ContextMenuItem>
            )}
            {authority === 'modify' && <ContextMenuSeparator />}
            <ContextMenuItem onClick={(e) => { e.stopPropagation(); onProperties(undefined, true); }} disabled={items?.is_single_file}>
              <Info className="mr-2 h-4 w-4" />
              Info
            </ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      )}
    </div>
  );
}
