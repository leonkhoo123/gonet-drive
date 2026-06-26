import { useRef, useEffect, useState, useCallback } from "react";
import { X, Trash2, Cloud, Share2, House, Pin, Pencil, Check, GripVertical, FolderOpen } from "lucide-react";
import { type StorageUsageResponse } from "@/api/api-file";
import { formatBytes } from "@/utils/utils";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useNavigate, useLocation } from "react-router-dom";
import { decodeUrlToPath, encodePathToUrl } from "@/utils/utils";
import { Logo } from "@/components/Logo";
import type { PinnedFolder } from "@/api/api-pinned";

function StorageIndicator({ 
  usage, 
  isWsConnected, 
  isHealthConnected 
}: { 
  usage?: StorageUsageResponse;
  isWsConnected: boolean;
  isHealthConnected: boolean;
}) {
  const lastUsage = useRef<StorageUsageResponse | undefined>(usage);

  useEffect(() => {
    if (usage) {
      lastUsage.current = usage;
    }
  }, [usage]);

  const displayUsage = usage ?? lastUsage.current;

  const indicators = (
    <div className="flex flex-col gap-1 items-end">
      <div className="flex items-center gap-1.5" title={`API: ${isHealthConnected ? 'OK' : 'Error'}`}>
        <span className="text-[10px] text-muted-foreground/80 font-mono tracking-wider leading-none">/health</span>
        <div className={`w-1.5 h-1.5 rounded-full ${isHealthConnected ? 'bg-green-500 shadow-[0_0_4px_#22c55e]' : 'bg-red-500 shadow-[0_0_4px_#ef4444]'}`} />
      </div>
      <div className="flex items-center gap-1.5" title={`WS: ${isWsConnected ? 'Connected' : 'Disconnected'}`}>
        <span className="text-[10px] text-muted-foreground/80 font-mono tracking-wider leading-none">WebSocket</span>
        <div className={`w-1.5 h-1.5 rounded-full ${isWsConnected ? 'bg-green-500 shadow-[0_0_4px_#22c55e]' : 'bg-red-500 shadow-[0_0_4px_#ef4444]'}`} />
      </div>
    </div>
  );

  if (!displayUsage) {
    return (
      <div className="px-4 py-4 border-b">
        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2 opacity-50">
          <Cloud className="h-4 w-4" />
          <span>Storage</span>
        </div>
        <Progress value={0} className="h-2 mb-2 bg-muted/50" />
        <div className="flex justify-between items-end">
          <div className="flex flex-col gap-1 text-xs text-muted-foreground opacity-50">
            <div><span className="font-medium">...</span> of <span className="font-medium">...</span> used</div>
            <div><span className="font-medium">...</span> left</div>
          </div>
          {indicators}
        </div>
      </div>
    );
  }

  const usedFormatted = formatBytes(displayUsage.used);
  
  if (displayUsage.limit <= 0) {
    return (
      <div className="px-4 py-4 border-b">
        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
          <Cloud className="h-4 w-4" />
          <span>Storage</span>
        </div>
        <div className="flex justify-between items-end">
          <div className="text-xs text-muted-foreground">
            <span className="font-medium text-foreground">{usedFormatted}</span> used
          </div>
          {indicators}
        </div>
      </div>
    );
  }

  const limitFormatted = formatBytes(displayUsage.limit);
  const leftFormatted = formatBytes(displayUsage.left || Math.max(0, displayUsage.limit - displayUsage.used));
  const percentage = Math.min(100, Math.max(0, (displayUsage.used / displayUsage.limit) * 100));

  return (
    <div className="px-4 py-4 border-b">
      <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
        <Cloud className="h-4 w-4" />
        <span>Storage</span>
      </div>
      <Progress value={percentage} className="h-2 mb-2 bg-muted/50" indicatorClassName={percentage > 90 ? "bg-gradient-to-r from-red-500 to-red-400" : "bg-gradient-to-r from-primary to-primary/80"} />
      <div className="flex justify-between items-end">
        <div className="flex flex-col gap-1 text-xs text-muted-foreground">
          <div><span className="font-medium text-foreground">{usedFormatted}</span> of <span className="font-medium text-foreground">{limitFormatted}</span> used</div>
          <div><span className="font-medium text-foreground">{leftFormatted}</span> left</div>
        </div>
        {indicators}
      </div>
    </div>
  );
}

interface HomeSidebarProps {
  isOpen: boolean;
  onClose: () => void;
  isWsConnected: boolean;
  isHealthConnected: boolean;
  titleName?: string;
  storageUsage?: StorageUsageResponse;
  pinnedFolders?: PinnedFolder[];
  isPinnedEditMode?: boolean;
  onTogglePinnedEditMode?: () => void;
  onUnpinFolder?: (path: string) => void;
  onReorderPinned?: (paths: string[]) => void;
}

// eslint-disable-next-line @typescript-eslint/no-empty-function
const noop = () => {};

export default function HomeSidebar({ isOpen, onClose, isWsConnected, isHealthConnected, titleName, storageUsage, pinnedFolders = [], isPinnedEditMode = false, onTogglePinnedEditMode = noop, onUnpinFolder = noop, onReorderPinned = noop }: HomeSidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const [isUnpinDialogOpen, setIsUnpinDialogOpen] = useState(false);
  const [unpinTarget, setUnpinTarget] = useState<string | null>(null);
  const [unpinLabel, setUnpinLabel] = useState('');

  useEffect(() => {
    if (!isOpen && isPinnedEditMode) {
      onTogglePinnedEditMode();
    }
  }, [isOpen, isPinnedEditMode, onTogglePinnedEditMode]);

  const itemRefs = useRef<Map<string, HTMLDivElement>>(new Map());
  const touchDrag = useRef<{ index: number; startY: number; currentIndex: number } | null>(null);
  const [touchOverIndex, setTouchOverIndex] = useState<number | null>(null);

  const getItemIndexFromY = useCallback((clientY: number): number | null => {
    for (let i = 0; i < pinnedFolders.length; i++) {
      const el = itemRefs.current.get(pinnedFolders[i].path);
      if (el) {
        const rect = el.getBoundingClientRect();
        if (clientY >= rect.top && clientY <= rect.bottom) {
          return i;
        }
      }
    }
    return null;
  }, [pinnedFolders]);

  const handleTouchStart = useCallback((index: number, e: React.TouchEvent) => {
    if (!isPinnedEditMode) return;
    const touch = e.touches[0];
    touchDrag.current = { index, startY: touch.clientY, currentIndex: index };
    setTouchOverIndex(index);
  }, [isPinnedEditMode]);

  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    if (!touchDrag.current) return;
    const touch = e.touches[0];
    touchDrag.current.currentIndex = touchDrag.current.index;
    const overIndex = getItemIndexFromY(touch.clientY);
    if (overIndex !== null && overIndex !== touchDrag.current.currentIndex) {
      setTouchOverIndex(overIndex);
    }
  }, [getItemIndexFromY]);

  const handleTouchEnd = useCallback(() => {
    if (!touchDrag.current) return;
    const { index, currentIndex: startIdx } = touchDrag.current;
    const endIndex = touchOverIndex;
    touchDrag.current = null;
    setTouchOverIndex(null);
    if (endIndex !== null && endIndex !== startIdx && endIndex !== index) {
      const reordered = [...pinnedFolders];
      const [moved] = reordered.splice(startIdx, 1);
      reordered.splice(endIndex, 0, moved);
      onReorderPinned(reordered.map((f) => f.path));
    }
  }, [pinnedFolders, onReorderPinned, touchOverIndex]);

  const handleNavigate = (path: string) => {
    if (window.innerWidth < 1024) {
      onClose();
      setTimeout(() => {
        void navigate(path);
      }, 300);
    } else {
      void navigate(path);
    }
  };

  const isActive = (path: string) => {
    let currentPath = decodeURIComponent(location.pathname.replace("/home", "")) || "/";
    currentPath = decodeUrlToPath(currentPath);
    return currentPath === path;
  };

  return (
    <>
      {/* Mobile Backdrop */}
      <div 
        className={`fixed inset-0 bg-background/80 backdrop-blur-sm lg:hidden transition-opacity duration-300
          ${isOpen ? "opacity-100 z-20" : "opacity-0 pointer-events-none z-0"}`}
        onClick={onClose}
      />

      {/* Sidebar/Drawer */}
      <aside 
        className={`
          fixed lg:relative z-30 h-full
          bg-gradient-to-b from-primary/[0.03] via-muted/10 to-muted/10 flex flex-col flex-shrink-0
          transition-all duration-300 ease-in-out overflow-hidden
          ${isOpen 
            ? "translate-x-0 w-72 border-r" 
            : "-translate-x-full w-72 lg:w-0 lg:translate-x-0 border-transparent lg:border-r-0"}
        `}
      >
        <div className="w-72 flex flex-col h-full overflow-hidden">
          <div className="px-4 py-3 border-b flex items-center justify-between shrink-0 h-16 md:h-14">
            <div 
              className="flex items-center gap-2 cursor-pointer"
              onClick={() => { handleNavigate("/home"); }}
            >
              <Logo className="w-8 h-8 object-contain shrink-0" />
              <h1 className="text-xl font-bold text-foreground tracking-tight">{titleName ?? "GoNet Drive"}</h1>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="icon" className="lg:hidden h-12 w-12 text-muted-foreground hover:text-foreground shrink-0" onClick={onClose}>
                <X className="h-6 w-6" />
              </Button>
            </div>
          </div>
          <StorageIndicator usage={storageUsage} isWsConnected={isWsConnected} isHealthConnected={isHealthConnected} />
          <div className="p-3 flex-1 overflow-auto space-y-1 pb-16">
            <div 
              className={`flex items-center gap-3 text-base md:text-sm px-3 py-3 md:py-2 rounded-md transition-colors cursor-pointer
                ${isActive("/") 
                  ? "bg-primary/10 text-primary shadow-[inset_0_0_0_1px_rgba(84,104,255,0.12)]" 
                  : "text-foreground hover:bg-muted/50"
                }`}
              onClick={() => { handleNavigate("/home"); }}
            >
              <House className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive("/") ? "text-primary" : "text-gray-500"}`} />
              <span className="truncate">Home</span>
            </div>

            <div 
              className={`flex items-center gap-3 text-base md:text-sm px-3 py-3 md:py-2 rounded-md transition-colors cursor-pointer
                ${isActive("/.cloud_delete") 
                  ? "bg-primary/10 text-primary shadow-[inset_0_0_0_1px_rgba(84,104,255,0.12)]" 
                  : "text-foreground hover:bg-muted/50"
                }`}
              onClick={() => { handleNavigate("/home/recycle_bin"); }}
            >
              <Trash2 className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive("/.cloud_delete") ? "text-primary" : "text-gray-500"}`} />
              <span className="truncate">Recycle Bin</span>
            </div>

            <div 
              className={`flex items-center gap-3 text-base md:text-sm px-3 py-3 md:py-2 rounded-md transition-colors cursor-pointer
                ${isActive("/manage-shares") 
                  ? "bg-primary/10 text-primary shadow-[inset_0_0_0_1px_rgba(84,104,255,0.12)]" 
                  : "text-foreground hover:bg-muted/50"
                }`}
              onClick={() => { handleNavigate("/manage-shares"); }}
            >
              <Share2 className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive("/manage-shares") ? "text-primary" : "text-gray-500"}`} />
              <span className="truncate">Manage Shares</span>
            </div>

            {pinnedFolders.length > 0 && (
              <>
                <div className="h-px bg-border mx-3 my-3" />
                <div className="flex items-center gap-2 px-3 pb-1">
                  <Pin className="h-4 w-4 text-gray-500 shrink-0" />
                  <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Pinned</span>
                  <div className="flex-1" />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={onTogglePinnedEditMode}
                    title={isPinnedEditMode ? "Done editing" : "Edit pinned folders"}
                  >
                    {isPinnedEditMode ? <Check className="h-3.5 w-3.5" /> : <Pencil className="h-3.5 w-3.5" />}
                  </Button>
                </div>
                {pinnedFolders.map((folder, index) => {
                  const basename = folder.path.split('/').filter(Boolean).pop() ?? folder.path;
                  const navPath = `/home${encodePathToUrl(folder.path)}`;
                  const isTouchOver = touchOverIndex === index;
                  return (
                    <div key={folder.path}
                      ref={(el) => {
                        if (el) itemRefs.current.set(folder.path, el);
                        else itemRefs.current.delete(folder.path);
                      }}
                      className={`flex items-center gap-3 text-base md:text-sm px-3 py-3 md:py-2 rounded-md transition-colors
                        ${!isPinnedEditMode ? 'cursor-pointer' : ''}
                        ${isTouchOver ? 'ring-2 ring-primary/50 bg-primary/5' : ''}
                        ${isActive(folder.path)
                          ? "bg-primary/10 text-primary shadow-[inset_0_0_0_1px_rgba(84,104,255,0.12)]"
                          : "text-foreground hover:bg-muted/50"
                        }`}
                      draggable={isPinnedEditMode}
                      onDragStart={(e) => {
                        if (!isPinnedEditMode) return;
                        e.dataTransfer.setData('text/plain', String(index));
                        e.dataTransfer.effectAllowed = 'move';
                      }}
                      onDragOver={(e) => {
                        if (!isPinnedEditMode) return;
                        e.preventDefault();
                        e.dataTransfer.dropEffect = 'move';
                      }}
                      onDragEnter={(e) => {
                        if (!isPinnedEditMode) return;
                        e.preventDefault();
                        setTouchOverIndex(index);
                      }}
                      onDragLeave={() => {
                        setTouchOverIndex(null);
                      }}
                      onDrop={(e) => {
                        if (!isPinnedEditMode) return;
                        e.preventDefault();
                        const fromIndex = Number(e.dataTransfer.getData('text/plain'));
                        if (fromIndex === index) return;
                        const reordered = [...pinnedFolders];
                        const [moved] = reordered.splice(fromIndex, 1);
                        reordered.splice(index, 0, moved);
                        onReorderPinned(reordered.map((f) => f.path));
                        setTouchOverIndex(null);
                      }}
                      onTouchStart={(e) => { handleTouchStart(index, e); }}
                      onTouchMove={handleTouchMove}
                      onTouchEnd={handleTouchEnd}
                      onClick={() => {
                        if (isPinnedEditMode) return;
                        handleNavigate(navPath);
                      }}
                    >
                      <FolderOpen className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive(folder.path) ? "text-primary" : "text-gray-500"}`} />
                      <span className="truncate flex-1">{basename}</span>
                      {isPinnedEditMode && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30 shrink-0"
                            onClick={(e) => {
                              e.stopPropagation();
                              setUnpinTarget(folder.path);
                              setUnpinLabel(basename);
                              setIsUnpinDialogOpen(true);
                            }}
                            title="Unpin"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                          <GripVertical className="h-4 w-4 text-gray-400 cursor-grab active:cursor-grabbing shrink-0" />
                        </>
                      )}
                    </div>
                  );
                })}
              </>
            )}

            {/* Audio Books - temporarily hidden */}
            {/* <div 
              className={`flex items-center gap-3 text-base md:text-sm px-3 py-3 md:py-2 rounded-md transition-colors cursor-pointer
                ${isActive("/audio-book") 
                  ? "bg-purple-100 dark:bg-purple-900/40 text-purple-600 dark:text-purple-400" 
                  : "text-foreground hover:bg-muted/50"
                }`}
              onClick={() => { handleNavigate("/audio-book"); }}
            >
              <BookAudio className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive("/audio-book") ? "text-purple-500" : "text-gray-500"}`} />
              <span className="truncate">Audio Books</span>
            </div> */}

            {/* Placeholder Items */}
            {/* <div className="flex items-center gap-3 text-base md:text-sm text-muted-foreground px-3 py-3 md:py-2 rounded-md hover:bg-muted/50 cursor-not-allowed transition-colors" title="Feature coming soon">
              <Folder className="h-5 w-5 md:h-4 md:w-4 text-primary/50 shrink-0" />
              <span className="truncate">Projects</span>
            </div>
            <div className="flex items-center gap-3 text-base md:text-sm text-muted-foreground px-3 py-3 md:py-2 rounded-md hover:bg-muted/50 cursor-not-allowed transition-colors" title="Feature coming soon">
              <Folder className="h-5 w-5 md:h-4 md:w-4 text-primary/50 shrink-0" />
              <span className="truncate">Documents</span>
            </div>
            <div className="flex items-center gap-3 text-base md:text-sm text-muted-foreground px-3 py-3 md:py-2 rounded-md hover:bg-muted/50 cursor-not-allowed transition-colors" title="Feature coming soon">
              <Folder className="h-5 w-5 md:h-4 md:w-4 text-primary/50 shrink-0" />
              <span className="truncate">Downloads</span>
            </div> */}
          </div>
        </div>
      </aside>

      <Dialog open={isUnpinDialogOpen} onOpenChange={setIsUnpinDialogOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Unpin folder?</DialogTitle>
            <DialogDescription>
              Remove "{unpinLabel}" from pinned folders?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setIsUnpinDialogOpen(false); }}>Cancel</Button>
            <Button variant="destructive" onClick={() => {
              if (unpinTarget) {
                onUnpinFolder(unpinTarget);
              }
              setIsUnpinDialogOpen(false);
              setUnpinTarget(null);
              setUnpinLabel('');
            }}>Unpin</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
