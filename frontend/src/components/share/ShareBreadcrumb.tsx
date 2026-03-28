import { Button } from "@/components/ui/button";
import { useNavigate, useParams } from "react-router-dom";
import { RefreshCcw, ArrowLeft, MoreVertical, Info, Download, Folder, File, Users } from "lucide-react";
import { encodePathToUrl } from "@/utils/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface ShareBreadcrumbProps {
  currentPath: string;
  shareRoot?: string;
  isFolderEmpty?: boolean;
  isSingleFile?: boolean;
  onProperties?: (fileName?: string, isCurrentDir?: boolean) => void;
  onRefresh?: () => void;
  onDownload?: () => void;
}

export default function ShareBreadcrumb({ 
  currentPath, 
  shareRoot,
  isFolderEmpty = false,
  isSingleFile = false,
  onProperties,
  onRefresh,
  onDownload,
}: ShareBreadcrumbProps) {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();

  return (
    <div className="h-14 border-b flex items-center justify-between px-2 md:px-6 bg-background shrink-0 gap-2">
      <div className="flex items-center gap-2 overflow-hidden">
        <div className="flex items-center text-sm text-muted-foreground overflow-hidden whitespace-nowrap">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => { void navigate(`/share/${id}/home${window.location.search}`); }}
            className="p-1 h-auto bg-transparent hover:bg-transparent hover:underline text-foreground shrink-0 hidden md:inline-flex items-center gap-1 font-semibold"
          >
            {isSingleFile ? (
              <File className="h-4 w-4 text-muted-foreground" />
            ) : (
              <Folder className="h-4 w-4 text-muted-foreground" />
            )}
            {isSingleFile ? "File Sharing" : (shareRoot ? (shareRoot.split("/").filter(Boolean).pop() ?? "Share Home") : "Share Home")}
            <Users className="h-3.5 w-3.5 text-muted-foreground ml-1.5" />
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
                      // Route navigation preserving search params
                      void navigate(`/share/${id}/home${encodePathToUrl(targetPath)}${window.location.search}`);
                    }}
                    className={`hover:underline hover:text-foreground transition-colors ${isHidden ? 'opacity-60' : ''}`}
                  >
                    {part}
                  </button>
                </span>
              );
            })}
          </div>
          <div className="md:hidden flex items-center shrink-0">
            {currentPath !== "/" && currentPath !== shareRoot ? (
              <Button
                variant="ghost"
                size="icon"
                onClick={() => {
                  const parts = currentPath.split("/").filter(Boolean);
                  parts.pop();
                  const targetPath = parts.length > 0 ? "/" + parts.join("/") : "";
                  void navigate(`/share/${id}/home${encodePathToUrl(targetPath)}${window.location.search}`);
                }}
                className="mr-1 h-12 w-12 md:h-8 md:w-8 text-muted-foreground hover:text-foreground shrink-0"
              >
                <ArrowLeft className="h-8 w-8 md:h-5 md:w-5" />
              </Button>
            ) : (
              <div className="mr-1 flex items-center justify-center h-12 w-12 md:h-8 md:w-8 text-muted-foreground shrink-0">
                {isSingleFile ? (
                  <File className="h-7 w-7 md:h-5 md:w-5" />
                ) : (
                  <Folder className="h-7 w-7 md:h-5 md:w-5" />
                )}
              </div>
            )}
            <div className="flex items-center gap-2">
              <span className="font-semibold text-foreground text-lg md:text-base flex items-center gap-1">
                {currentPath === "/" || currentPath === shareRoot
                  ? (isSingleFile ? "File Sharing" : (shareRoot ? (shareRoot.split("/").filter(Boolean).pop() ?? "Share Home") : "Share Home")) 
                  : currentPath.split("/").filter(Boolean).pop()}
                {(currentPath === "/" || currentPath === shareRoot) && (
                  <Users className="h-4 w-4 md:h-3.5 md:w-3.5 text-muted-foreground ml-1.5" />
                )}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
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
              <DropdownMenuItem onClick={() => onDownload?.()} disabled={isFolderEmpty}>
                <Download className="mr-2 h-4 w-4" />
                Download
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onProperties?.(undefined, true)}>
                <Info className="mr-2 h-4 w-4" />
                Info
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  );
}
