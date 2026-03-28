import { useRef, useEffect } from "react";
import { X, Trash2, Cloud, BookAudio, Share2 } from "lucide-react";
import { type StorageUsageResponse } from "@/api/api-file";
import { formatBytes } from "@/utils/utils";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { useNavigate, useLocation } from "react-router-dom";
import { decodeUrlToPath } from "@/utils/utils";
import { getConfig } from "@/config";

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
      <Progress value={percentage} className="h-2 mb-2 bg-muted/50" indicatorClassName={percentage > 90 ? "bg-red-500" : "bg-primary"} />
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
}

export default function HomeSidebar({ isOpen, onClose, isWsConnected, isHealthConnected, titleName, storageUsage }: HomeSidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const handleNavigate = (path: string) => {
    void navigate(path);
    if (window.innerWidth < 1024) {
      onClose();
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
      {isOpen && (
        <div 
          className="fixed inset-0 z-20 bg-background/80 backdrop-blur-sm lg:hidden"
          onClick={onClose}
        />
      )}

      {/* Sidebar/Drawer */}
      <aside 
        className={`
          fixed lg:relative z-30 h-full
          bg-muted/10 flex flex-col flex-shrink-0
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
              <img src={`${getConfig().apiBaseUrl}/config/logo`} alt="Logo" className="w-10 h-10 object-contain" />
              <h1 className="text-xl font-bold text-foreground tracking-tight">{titleName ?? "Cloud Drive"}</h1>
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
                ${isActive("/.cloud_delete") 
                  ? "bg-primary/10 text-primary" 
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
                  ? "bg-primary/10 text-primary" 
                  : "text-foreground hover:bg-muted/50"
                }`}
              onClick={() => { handleNavigate("/manage-shares"); }}
            >
              <Share2 className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive("/manage-shares") ? "text-primary" : "text-gray-500"}`} />
              <span className="truncate">Manage Shares</span>
            </div>

            <div 
              className={`flex items-center gap-3 text-base md:text-sm px-3 py-3 md:py-2 rounded-md transition-colors cursor-pointer
                ${isActive("/audio-book") 
                  ? "bg-purple-100 dark:bg-purple-900/40 text-purple-600 dark:text-purple-400" 
                  : "text-foreground hover:bg-muted/50"
                }`}
              onClick={() => { handleNavigate("/audio-book"); }}
            >
              <BookAudio className={`h-5 w-5 md:h-4 md:w-4 shrink-0 ${isActive("/audio-book") ? "text-purple-500" : "text-gray-500"}`} />
              <span className="truncate">Audio Books</span>
            </div>

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
    </>
  );
}
