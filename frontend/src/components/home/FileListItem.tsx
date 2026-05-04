import { memo, useState } from "react";
import { Trash2, Folder, Pencil, Trash2 as TrashIcon, Download, Info, Share2, MoreVertical, Scissors, Copy, Clipboard, Users, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatBytes, formatLastModified } from "@/utils/utils";
import type { FileInterface } from "@/api/api-file";
import { getFileIcon } from "@/utils/fileIcon";
import { TruncatedText } from "@/components/custom/truncatedText";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";

interface FileListItemProps {
  file: FileInterface;
  index: number;
  isSelected: boolean;
  isHidden: boolean;
  isCut: boolean;
  isRecycleBin: boolean;
  isTouchDevice: boolean;
  transitioningFolder: string | null;
  openDropdownName: string | null;
  setOpenDropdownName: (name: string | null) => void;
  handleItemClick: (file: FileInterface, index: number, e: React.MouseEvent) => void;
  handleItemDoubleClick: (file: FileInterface) => void;
  handleTouchStart: (file: FileInterface, index: number) => void;
  handleTouchEnd: () => void;
  handleTouchMove: () => void;
  onRename: (fileName?: string) => void;
  onDelete: (fileName?: string) => void;
  onDownload: (fileName?: string) => void;
  onProperties: (fileName?: string) => void;
  onShare?: (fileName?: string, isCurrentDir?: boolean) => void;
  onCut: () => void;
  onCopy: () => void;
  onPaste: () => void;
  onFileContextMenu: (fileInfo: FileInterface, index: number) => void;
  clipboardItemsCount: number;
  clipboardOperation: 'cut' | 'copy' | null;
  clipboardSourceDir?: string;
  currentPath: string;
  selectedItemsSize: number;
  hasSelectedDelete: boolean; // if selectedItems.has('.cloud_delete')
  viewMode?: 'list' | 'grid';
}

export const FileListItem = memo(function FileListItem({
  file,
  index,
  isSelected,
  isHidden,
  isCut,
  isRecycleBin,
  isTouchDevice,
  transitioningFolder,
  openDropdownName,
  setOpenDropdownName,
  handleItemClick,
  handleItemDoubleClick,
  handleTouchStart,
  handleTouchEnd,
  handleTouchMove,
  onRename,
  onDelete,
  onDownload,
  onProperties,
  onShare,
  onCut,
  onCopy,
  onPaste,
  onFileContextMenu,
  clipboardItemsCount,
  clipboardOperation,
  clipboardSourceDir,
  currentPath,
  selectedItemsSize,
  hasSelectedDelete,
  viewMode = 'list',
}: FileListItemProps) {
  const [imgErrorUrl, setImgErrorUrl] = useState<string | null>(null);

  const fileContent = viewMode === 'grid' ? (
    <div id={`file-item-${index}`} className="group h-full w-full">
      <div
        onClick={(e) => { handleItemClick(file, index, e); }}
        onDoubleClick={() => { handleItemDoubleClick(file); }}
        onTouchStart={() => { handleTouchStart(file, index); }}
        onTouchEnd={handleTouchEnd}
        onTouchMove={handleTouchMove}
        onTouchCancel={handleTouchEnd}
        onContextMenu={(e) => {
          if (window.matchMedia("(pointer: coarse)").matches) { e.preventDefault(); }
        }}
        className={`flex flex-col items-center p-3 cursor-pointer rounded-lg transition-all duration-75 ease-out select-none relative w-full h-full border [-webkit-touch-callout:none] [-webkit-tap-highlight-color:transparent] ${
          transitioningFolder === file.name
            ? 'bg-primary/30 border-primary/50'
            : isSelected
              ? 'bg-primary/10 border-primary/30 @media(hover:hover):hover:bg-primary/20'
              : 'bg-card border-transparent @media(hover:hover):hover:bg-muted/50 @media(hover:hover):hover:border-border'
        } ${isCut ? 'opacity-50' : ''}`}
      >
        <div className={`w-full aspect-square flex items-center justify-center mb-3 overflow-hidden rounded-md bg-muted/20 ${isHidden ? 'opacity-60' : ''}`}>
          {file.type === "dir" ? (
            file.name === ".cloud_delete" ? (
              <Trash2 className="h-16 w-16 text-primary" />
            ) : (
              <Folder className="h-16 w-16 text-primary fill-primary/20" />
            )
          ) : (
            file.media_type === "photo" && imgErrorUrl !== file.url ? (
              <img 
                src={file.url.replace('/photo/play/', '/photo/thumbnail/')} 
                loading="lazy" 
                alt={file.name} 
                className="object-contain w-full h-full transition-opacity duration-300"
                onLoad={(e) => { e.currentTarget.style.opacity = '1'; }}
                style={{ opacity: 0 }}
                onError={() => { setImgErrorUrl(file.url); }}
              />
            ) : (
              <div className="[&>svg]:w-16 [&>svg]:h-16">
                {getFileIcon(file)}
              </div>
            )
          )}
        </div>

        <div className={`w-full flex flex-col items-center text-center ${isHidden ? 'opacity-60' : ''}`}>
          <div className="flex items-center justify-center space-x-1 w-full px-1 relative">
            <TruncatedText className="text-sm font-medium text-foreground w-full" text={file.name === ".cloud_delete" ? "Recycle Bin" : file.name} />
          </div>
          {file.type !== "dir" && (
            <div className="text-xs text-muted-foreground mt-1 opacity-80">
              {formatBytes(file.size)}
            </div>
          )}
        </div>

        {file.isShared && <Users className="h-4 w-4 text-muted-foreground shrink-0 absolute top-2 left-2 bg-background/50 rounded-full p-0.5" />}

        {selectedItemsSize === 0 && (
          <div className="md:hidden absolute top-1 right-1">
            <DropdownMenu 
              open={openDropdownName === file.name}
              onOpenChange={(open) => {
                if (!open) setOpenDropdownName(null);
              }}
            >
              <DropdownMenuTrigger asChild>
                <div className="absolute inset-0 pointer-events-none" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48" onClick={(e) => { e.stopPropagation(); }}>
                <DropdownMenuItem disabled={isRecycleBin || file.name === '.cloud_delete'} onClick={(e) => { 
                  e.stopPropagation(); setOpenDropdownName(null); onRename(file.name);
                }}>
                  <Pencil className="mr-2 h-4 w-4" /> Rename
                </DropdownMenuItem>
                <DropdownMenuItem disabled={file.name === '.cloud_delete'} onClick={(e) => { 
                  e.stopPropagation(); setOpenDropdownName(null); onDelete(file.name);
                }} className="text-red-600 focus:bg-red-50 focus:text-red-600 dark:focus:bg-red-950/30">
                  <TrashIcon className="mr-2 h-4 w-4" /> Delete
                </DropdownMenuItem>
                <DropdownMenuItem disabled={isRecycleBin} onClick={(e) => { 
                  e.stopPropagation(); setOpenDropdownName(null); onDownload(file.name);
                }}>
                  <Download className="mr-2 h-4 w-4" /> Download
                </DropdownMenuItem>
                <DropdownMenuItem onClick={(e) => { 
                  e.stopPropagation(); setOpenDropdownName(null); onProperties(file.name);
                }}>
                  <Info className="mr-2 h-4 w-4" /> Info
                </DropdownMenuItem>
                {!isRecycleBin && (
                  <DropdownMenuItem onClick={(e) => {
                    e.stopPropagation(); setOpenDropdownName(null); onShare?.(file.name, false);
                  }}>
                    <Share2 className="mr-2 h-4 w-4" /> Share
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>

            <Button 
              variant="ghost" 
              size="icon" 
              className="h-8 w-8 text-muted-foreground bg-background/50 hover:bg-background/80 rounded-full" 
              onClick={(e) => { 
                e.stopPropagation(); 
                setOpenDropdownName(openDropdownName === file.name ? null : file.name);
              }}
            >
              <MoreVertical className="h-4 w-4" />
            </Button>
          </div>
        )}
      </div>
    </div>
  ) : (
    <div id={`file-item-${index}`} className="group">
      <div
        onClick={(e) => {
          handleItemClick(file, index, e);
        }}
        onDoubleClick={() => { handleItemDoubleClick(file); }}
        onTouchStart={() => { handleTouchStart(file, index); }}
        onTouchEnd={handleTouchEnd}
        onTouchMove={handleTouchMove}
        onTouchCancel={handleTouchEnd}
        onContextMenu={(e) => {
          if (window.matchMedia("(pointer: coarse)").matches) {
            e.preventDefault();
          }
        }}
        className={`flex items-center pl-4 pr-1 md:pl-1 md:pr-3 py-2 md:py-3 cursor-pointer rounded-md transition-all duration-75 ease-out select-none min-h-[64px] md:min-h-[44px] [-webkit-touch-callout:none] [-webkit-tap-highlight-color:transparent] ${
          transitioningFolder === file.name
            ? 'bg-primary/30'
            : isSelected
              ? 'bg-primary/10 @media(hover:hover):hover:bg-primary/20'
              : 'bg-transparent @media(hover:hover):hover:bg-muted/50'
        } ${isCut ? 'opacity-50' : ''}`}
      >
        {/* NAME */}
        <div className={`flex-1 flex items-center space-x-3 min-w-0 pr-2 md:pr-4 ${isHidden ? 'opacity-60' : ''}`}>
          {file.type === "dir" ? (
            <>
              {file.name === ".cloud_delete" ? (
                <Trash2 className="h-7 w-7 md:h-5 md:w-5 shrink-0 text-primary" />
              ) : (
                <Folder className="h-7 w-7 md:h-5 md:w-5 shrink-0 text-primary fill-primary/20" />
              )}
              <div className="flex flex-col pl-1 min-w-0 w-full text-left">
                <div className="flex items-center space-x-2 min-w-0 w-full">
                  <TruncatedText className="font-medium text-foreground" text={file.name === ".cloud_delete" ? "Recycle Bin" : file.name} />
                  {file.isShared && <Users className="h-3.5 w-3.5 text-muted-foreground shrink-0" />}
                </div>
                <TruncatedText className="text-xs mt-0.5 lg:hidden text-muted-foreground" text={formatLastModified(file.modified)} />
              </div>
            </>
          ) : (
            <>
              {file.media_type === "photo" && imgErrorUrl !== file.url ? (
                <div className="h-10 w-10 md:h-8 md:w-8 shrink-0 rounded overflow-hidden bg-muted/50 flex items-center justify-center relative">
                  <img 
                    src={file.url.replace('/photo/play/', '/photo/thumbnail/')} 
                    loading="lazy" 
                    alt={file.name} 
                    className="object-contain w-full h-full transition-opacity duration-300"
                    onLoad={(e) => { e.currentTarget.style.opacity = '1'; }}
                    style={{ opacity: 0 }}
                    onError={() => { setImgErrorUrl(file.url); }}
                  />
                </div>
              ) : (
                getFileIcon(file)
              )}
              <div className="flex flex-col pl-1 min-w-0 w-full text-left">
                <div className="flex items-center space-x-2 min-w-0 w-full">
                  <TruncatedText className="text-foreground" text={file.name} />
                  {file.isShared && <Users className="h-3.5 w-3.5 text-muted-foreground shrink-0" />}
                </div>
                <TruncatedText className="text-xs mt-0.5 lg:hidden text-muted-foreground" text={`${formatBytes(file.size)} • ${formatLastModified(file.modified)}`} />
              </div>
            </>
          )}
        </div>

        {/* SIZE (desktop only) */}
        <div className={`w-24 md:w-32 hidden lg:block text-right text-sm text-muted-foreground ${isHidden ? 'opacity-60' : ''}`}>
          {file.type === "dir" ? "--" : formatBytes(file.size)}
        </div>

        {/* LAST MODIFIED (desktop only) */}
        <div className={`w-32 md:w-48 hidden lg:block text-right text-sm text-muted-foreground ${isHidden ? 'opacity-60' : ''}`}>
          {formatLastModified(file.modified)}
        </div>

        {/* ACTIONS (mobile only) */}
        {selectedItemsSize === 0 && (
          <div className="md:hidden ml-1 flex items-center shrink-0 relative">
            <DropdownMenu 
              open={openDropdownName === file.name}
              onOpenChange={(open) => {
                if (!open) setOpenDropdownName(null);
              }}
            >
              <DropdownMenuTrigger asChild>
                <div className="absolute inset-0 pointer-events-none" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48" onClick={(e) => { e.stopPropagation(); }}>
                <DropdownMenuItem disabled={isRecycleBin || file.name === '.cloud_delete'} onClick={(e) => { 
                  e.stopPropagation(); 
                  setOpenDropdownName(null);
                  onRename(file.name);
                }}>
                  <Pencil className="mr-2 h-4 w-4" />
                  Rename
                </DropdownMenuItem>
                <DropdownMenuItem disabled={file.name === '.cloud_delete'} onClick={(e) => { 
                  e.stopPropagation(); 
                  setOpenDropdownName(null);
                  onDelete(file.name);
                }} className="text-red-600 focus:bg-red-50 focus:text-red-600 dark:focus:bg-red-950/30">
                  <TrashIcon className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
                <DropdownMenuItem disabled={isRecycleBin} onClick={(e) => { 
                  e.stopPropagation(); 
                  setOpenDropdownName(null);
                  onDownload(file.name);
                }}>
                  <Download className="mr-2 h-4 w-4" />
                  Download
                </DropdownMenuItem>
                <DropdownMenuItem onClick={(e) => { 
                  e.stopPropagation(); 
                  setOpenDropdownName(null);
                  onProperties(file.name);
                }}>
                  <Info className="mr-2 h-4 w-4" />
                  Info
                </DropdownMenuItem>
                {!isRecycleBin && (
                  <DropdownMenuItem onClick={(e) => {
                    e.stopPropagation();
                    setOpenDropdownName(null);
                    onShare?.(file.name, false);
                  }}>
                    <Share2 className="mr-2 h-4 w-4" />
                    Share
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>

            <Button 
              variant="ghost" 
              size="icon" 
              className="h-10 w-8 text-muted-foreground" 
              onClick={(e) => { 
                e.stopPropagation(); 
                setOpenDropdownName(openDropdownName === file.name ? null : file.name);
              }}
            >
              <MoreVertical className="h-5 w-5" />
            </Button>
          </div>
        )}
      </div>
    </div>
  );

  if (isTouchDevice) {
    return (
      <div key={file.name}>
        {fileContent}
      </div>
    );
  }

  const canOpen = file.type === "dir" || ["video", "photo", "music", "text_documents", "pdf"].includes(file.media_type ?? "");

  return (
    <ContextMenu key={file.name} onOpenChange={(open) => {
      if (open) {
        onFileContextMenu(file, index);
      }
    }}>
      <ContextMenuTrigger asChild>
        {fileContent}
      </ContextMenuTrigger>
      <ContextMenuContent className="w-64">
        {canOpen && (
          <>
            <ContextMenuItem onClick={(e) => { e.stopPropagation(); handleItemDoubleClick(file); }} disabled={selectedItemsSize > 1 || hasSelectedDelete}>
              <ExternalLink className="mr-2 h-4 w-4" />
              Open
            </ContextMenuItem>
            <ContextMenuSeparator />
          </>
        )}
        <ContextMenuItem onClick={(e) => { e.stopPropagation(); onCut(); }} disabled={selectedItemsSize === 0 || hasSelectedDelete}>
          <Scissors className="mr-2 h-4 w-4" />
          Cut
        </ContextMenuItem>
        <ContextMenuItem onClick={(e) => { e.stopPropagation(); onCopy(); }} disabled={selectedItemsSize === 0 || isRecycleBin || hasSelectedDelete}>
          <Copy className="mr-2 h-4 w-4" />
          Copy
        </ContextMenuItem>
        <ContextMenuItem 
          onClick={(e) => { e.stopPropagation(); onPaste(); }} 
          disabled={clipboardItemsCount === 0 || (clipboardOperation === 'cut' && clipboardSourceDir === currentPath) || isRecycleBin}
        >
          <Clipboard className="mr-2 h-4 w-4" />
          Paste
        </ContextMenuItem>
        <ContextMenuSeparator />
        {selectedItemsSize <= 1 && (
          <ContextMenuItem onClick={(e) => { e.stopPropagation(); onRename(file.name); }} disabled={selectedItemsSize !== 1 || isRecycleBin || hasSelectedDelete}>
            <Pencil className="mr-2 h-4 w-4" />
            Rename
          </ContextMenuItem>
        )}
        <ContextMenuItem onClick={(e) => { e.stopPropagation(); onDelete(file.name); }} disabled={selectedItemsSize === 0 || hasSelectedDelete} className="text-red-600 focus:bg-red-50 focus:text-red-600 dark:focus:bg-red-950/30">
          <TrashIcon className="mr-2 h-4 w-4" />
          Delete
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem onClick={(e) => { e.stopPropagation(); onDownload(file.name); }} disabled={isRecycleBin || selectedItemsSize === 0 || hasSelectedDelete}>
          <Download className="mr-2 h-4 w-4" />
          Download
        </ContextMenuItem>
        <ContextMenuItem onClick={(e) => { e.stopPropagation(); onProperties(file.name); }} disabled={selectedItemsSize === 0}>
          <Info className="mr-2 h-4 w-4" />
          Info
        </ContextMenuItem>
        {!isRecycleBin && selectedItemsSize <= 1 && (
          <ContextMenuItem onClick={(e) => { e.stopPropagation(); onShare?.(file.name); }} disabled={selectedItemsSize === 0 || hasSelectedDelete}>
            <Share2 className="mr-2 h-4 w-4" />
            Share
          </ContextMenuItem>
        )}
      </ContextMenuContent>
    </ContextMenu>
  );
}, (prevProps, nextProps) => {
  return (
    prevProps.file.name === nextProps.file.name &&
    prevProps.file.modified === nextProps.file.modified &&
    prevProps.isSelected === nextProps.isSelected &&
    prevProps.isHidden === nextProps.isHidden &&
    prevProps.isCut === nextProps.isCut &&
    prevProps.isRecycleBin === nextProps.isRecycleBin &&
    prevProps.isTouchDevice === nextProps.isTouchDevice &&
    prevProps.transitioningFolder === nextProps.transitioningFolder &&
    prevProps.openDropdownName === nextProps.openDropdownName &&
    prevProps.clipboardItemsCount === nextProps.clipboardItemsCount &&
    prevProps.clipboardOperation === nextProps.clipboardOperation &&
    prevProps.clipboardSourceDir === nextProps.clipboardSourceDir &&
    prevProps.currentPath === nextProps.currentPath &&
    prevProps.selectedItemsSize === nextProps.selectedItemsSize &&
    prevProps.hasSelectedDelete === nextProps.hasSelectedDelete &&
    prevProps.index === nextProps.index
  );
});
