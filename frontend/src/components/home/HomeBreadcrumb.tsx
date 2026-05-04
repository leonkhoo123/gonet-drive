import { Button } from "@/components/ui/button";
import { useNavigate } from "react-router-dom";
import { Menu, LogOut, RefreshCcw, Settings, Sun, Moon, Monitor, Eye, EyeOff, ArrowLeft, MoreVertical, Info, Sliders, Trash2, Download, Share2 } from "lucide-react";
import { encodePathToUrl } from "@/utils/utils";
import { logout, getMe } from "@/api/api-auth";
import { useTheme } from "@/components/theme-provider";
import { toast } from "sonner";
import { useRegisterSW } from "virtual:pwa-register/react";
import { usePreferences } from "@/context/PreferencesContext";
import { useState, useEffect } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { DevicesModal } from "@/components/custom/DevicesModal";
import { MonitorSmartphone } from "lucide-react";

interface HomeBreadcrumbProps {
  currentPath: string;
  isFolderEmpty?: boolean;
  onToggleSidebar?: () => void;
  onProperties?: (fileName?: string, isCurrentDir?: boolean) => void;
  onShare?: (fileName?: string, isCurrentDir?: boolean) => void;
  onRefresh?: () => void;
  onDownload?: () => void;
  onEmptyRecycleBin?: () => void;
}

export default function HomeBreadcrumb({ 
  currentPath, 
  isFolderEmpty = false,
  onToggleSidebar,
  onProperties,
  onShare,
  onRefresh,
  onDownload,
  onEmptyRecycleBin,
}: HomeBreadcrumbProps) {
  const navigate = useNavigate();
  const { updateServiceWorker } = useRegisterSW();
  const { theme, setTheme } = useTheme();
  const { showHidden, setShowHidden } = usePreferences();
  const [isAdmin, setIsAdmin] = useState(false);
  const [devicesModalOpen, setDevicesModalOpen] = useState(false);

  useEffect(() => {
    getMe().then((res) => {
      setIsAdmin(res.role === "admin" || res.role === "superadmin");
    }).catch(console.error);
  }, []);

  const handleReload = async () => {
    console.log("Force reloaded");
    await updateServiceWorker(true); // force service worker update + reload
    toast.success("App Reloaded");
  };

  const handleLogout = async () => {
    await logout();
    toast.success("Logged Out");
    console.log("Logged Out");
    void navigate("/login");
  };

  const toggleTheme = () => {
    if (theme === "light") {
      setTheme("dark");
    } else if (theme === "dark") {
      setTheme("system");
    } else {
      setTheme("light");
    }
  };

  return (
    <div className="h-16 md:h-14 border-b flex items-center justify-between px-2 md:px-6 bg-background shrink-0 gap-2">
      <div className="flex items-center gap-2 overflow-hidden">
        {onToggleSidebar && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggleSidebar}
            className={`mr-1 h-12 w-12 md:h-8 md:w-8 text-muted-foreground hover:text-foreground shrink-0 ${currentPath !== "/" ? "hidden md:flex" : ""}`}
          >
            <Menu className="h-8 w-8 md:h-5 md:w-5" />
          </Button>
        )}
        <div className="flex items-center text-sm text-muted-foreground overflow-hidden whitespace-nowrap">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => { void navigate("/home"); }}
            className="p-1 h-auto bg-transparent hover:bg-transparent hover:underline text-foreground shrink-0 hidden md:inline-flex"
          >
            Home
          </Button>
          <div className="hidden md:flex items-center">
            {currentPath.split("/").filter(Boolean).map((part, idx, arr) => {
              const isHidden = part.startsWith('.');
              return (
                <span key={idx} className="flex items-center shrink-0">
                  <span className="mx-1 text-muted-foreground/50">/</span>
                  <button
                    onClick={() => {
                      const targetPath = "/" + arr.slice(0, idx + 1).join("/");
                      void navigate("/home" + encodePathToUrl(targetPath));
                    }}
                    className={`hover:underline hover:text-foreground transition-colors ${isHidden ? 'opacity-60' : ''}`}
                  >
                    {part === '.cloud_delete' ? 'Recycle Bin' : part}
                  </button>
                </span>
              );
            })}
          </div>
          <div className="md:hidden flex items-center shrink-0">
            {currentPath !== "/" && (
              <Button
                variant="ghost"
                size="icon"
                onClick={() => {
                  const parts = currentPath.split("/").filter(Boolean);
                  parts.pop();
                  const targetPath = "/" + parts.join("/");
                  void navigate("/home" + encodePathToUrl(targetPath));
                }}
                className="mr-1 h-12 w-12 md:h-8 md:w-8 text-muted-foreground hover:text-foreground shrink-0"
              >
                <ArrowLeft className="h-8 w-8 md:h-5 md:w-5" />
              </Button>
            )}
            <span className="font-semibold text-foreground text-lg md:text-base">
              {currentPath === "/" 
                ? "Home" 
                : currentPath.split("/").filter(Boolean).pop() === '.cloud_delete' 
                  ? 'Recycle Bin' 
                  : currentPath.split("/").filter(Boolean).pop()}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        {/* Settings Button */}
        <div className={currentPath !== "/" ? "hidden md:flex" : "flex"}>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon" className="h-12 w-12 md:h-9 md:w-9">
                <Settings className="h-8 w-8 md:h-[1.2rem] md:w-[1.2rem] transition-all" />
                <span className="sr-only">Settings</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              
              <DropdownMenuItem onSelect={(e) => {
                e.preventDefault();
                toggleTheme();
              }}>
                {theme === "light" && <Sun className="mr-2 h-4 w-4" />}
                {theme === "dark" && <Moon className="mr-2 h-4 w-4" />}
                {theme === "system" && <Monitor className="mr-2 h-4 w-4" />}
                <span>Theme: {theme.charAt(0).toUpperCase() + theme.slice(1)}</span>
              </DropdownMenuItem>

              <DropdownMenuItem onClick={() => { setShowHidden(!showHidden); }}>
                {showHidden ? (
                  <Eye className="mr-2 h-4 w-4" />
                ) : (
                  <EyeOff className="mr-2 h-4 w-4" />
                )}
                <span>{showHidden ? "Hide Hidden Files" : "Show Hidden Files"}</span>
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              <DropdownMenuItem onClick={() => { setDevicesModalOpen(true); }}>
                <MonitorSmartphone className="mr-2 h-4 w-4" />
                <span>Devices</span>
              </DropdownMenuItem>

              {isAdmin && (
                <DropdownMenuItem onClick={() => { void navigate("/admin"); }}>
                  <Sliders className="mr-2 h-4 w-4" />
                  <span>Manage Cloud</span>
                </DropdownMenuItem>
              )}

              <DropdownMenuItem onClick={handleReload}>
                <RefreshCcw className="mr-2 h-4 w-4" />
                <span>Reload App</span>
              </DropdownMenuItem>

              <DropdownMenuItem onClick={handleLogout} className="text-red-600 focus:text-red-600 focus:bg-red-100 dark:focus:bg-red-900/30">
                <LogOut className="mr-2 h-4 w-4" />
                <span>Log Out</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* Mobile Action Buttons */}
        <div className="md:hidden flex items-center">
          <Button variant="ghost" size="icon" className="h-12 w-12 text-muted-foreground" onClick={onRefresh}>
            <RefreshCcw className="h-6 w-6" />
            <span className="sr-only">Refresh</span>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-12 w-12 text-muted-foreground">
                <MoreVertical className="h-6 w-6" />
                <span className="sr-only">More options</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuItem onClick={() => onDownload?.()} disabled={isFolderEmpty || currentPath.split("/").filter(Boolean).pop() === '.cloud_delete'}>
                <Download className="mr-2 h-4 w-4" />
                Download All
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onProperties?.(undefined, true)}>
                <Info className="mr-2 h-4 w-4" />
                Info
              </DropdownMenuItem>
              {currentPath !== "/" && (
                <DropdownMenuItem onClick={() => onShare?.(undefined, true)}>
                  <Share2 className="mr-2 h-4 w-4" />
                  Share
                </DropdownMenuItem>
              )}
              {currentPath.split("/").filter(Boolean).pop() === '.cloud_delete' && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={onEmptyRecycleBin}>
                    <Trash2 className="mr-2 h-4 w-4 text-red-600" />
                    <span className="text-red-600">Empty Recycle Bin</span>
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <DevicesModal 
        open={devicesModalOpen} 
        onOpenChange={setDevicesModalOpen} 
      />
    </div>
  );
}
