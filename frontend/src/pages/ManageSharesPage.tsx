import { useEffect, useState, useMemo, useCallback } from "react";
import { getShares, toggleShareBlock, deleteShare, isNeverExpires } from "@/api/api-share";
import type { ShareItem } from "@/api/api-share";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Trash2, Ban, CheckCircle, Share2, ArrowLeft, Folder, FileText, ChevronDown, ChevronUp } from "lucide-react";
import { useNavigate } from "react-router-dom";
import DefaultLayout from "@/layouts/DefaultLayout";
import HomeSidebar from "@/components/home/HomeSidebar";
import ManageSharesDeleteDialog from "@/components/share/ManageSharesDeleteDialog";
import { useAppHealth } from "@/hooks/useAppHealth";
import { fetchDirList, type StorageUsageResponse } from "@/api/api-file";

export default function ManageSharesPage() {
  const [shares, setShares] = useState<ShareItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});
  const navigate = useNavigate();

  const [isSidebarOpen, setIsSidebarOpen] = useState(window.innerWidth >= 1024);
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [storageUsage, setStorageUsage] = useState<StorageUsageResponse | undefined>(undefined);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [shareToDelete, setShareToDelete] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleAppRefresh = useCallback(() => {
    setRefreshTrigger(prev => prev + 1);
  }, []);

  const { healthData, isWsConnected, isHealthConnected } = useAppHealth(handleAppRefresh);

  useEffect(() => {
    const fetchStorage = async () => {
      try {
        const rs = await fetchDirList("/", false, undefined, undefined);
        setStorageUsage(rs.storage);
      } catch (error) {
        console.error("Failed to fetch storage usage", error);
      }
    };
    void fetchStorage();
  }, [refreshTrigger]);

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

  const fetchShares = async () => {
    try {
      setLoading(true);
      const data = await getShares();
      setShares(data);
    } catch (error) {
      console.error(error);
      toast.error("Failed to load shares");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchShares();
  }, []);

  const handleToggleBlock = async (id: string, currentStatus: boolean) => {
    try {
      await toggleShareBlock(id);
      toast.success(currentStatus ? "Share unblocked" : "Share blocked");
      void fetchShares();
    } catch (error) {
      console.error(error);
      toast.error("Failed to update share status");
    }
  };

  const handleDelete = async () => {
    if (!shareToDelete) return;
    try {
      setIsDeleting(true);
      await deleteShare(shareToDelete);
      toast.success("Share deleted");
      setIsDeleteDialogOpen(false);
      setShareToDelete(null);
      void fetchShares();
    } catch (error) {
      console.error(error);
      toast.error("Failed to delete share");
    } finally {
      setIsDeleting(false);
    }
  };

  const openDeleteDialog = (id: string) => {
    setShareToDelete(id);
    setIsDeleteDialogOpen(true);
  };

  const togglePath = (path: string) => {
    setExpandedPaths((prev) => ({ ...prev, [path]: !prev[path] }));
  };

  const groupedShares = useMemo(() => {
    const groups: Record<string, ShareItem[]> = {};
    for (const share of shares) {
      if (!(share.path in groups)) {
        groups[share.path] = [];
      }
      groups[share.path].push(share);
    }
    return groups;
  }, [shares]);

  return (
    <DefaultLayout>
      <div className="flex flex-1 w-full overflow-hidden relative">
        <HomeSidebar 
          isOpen={isSidebarOpen} 
          onClose={() => { setIsSidebarOpen(false); }} 
          isWsConnected={isWsConnected}
          isHealthConnected={isHealthConnected}
          titleName={healthData?.service_name}
          storageUsage={storageUsage}
        />

        <div className="flex-1 flex flex-col min-w-0 bg-background overflow-hidden">
          <header className="h-16 md:h-14 border-b flex items-center px-2 md:px-6 shrink-0 gap-2">
            <Button variant="ghost" size="icon" onClick={() => navigate("/home")} className="lg:hidden h-10 w-10 shrink-0 text-muted-foreground hover:text-foreground">
              <ArrowLeft className="h-5 w-5" />
            </Button>
            <div className="hidden lg:block">
              <Button variant="ghost" size="icon" onClick={() => navigate("/home")} className="h-8 w-8 text-muted-foreground hover:text-foreground shrink-0">
                <ArrowLeft className="h-5 w-5" />
              </Button>
            </div>
            <div className="flex items-center gap-2 lg:ml-2">
              <Share2 className="h-6 w-6 md:h-5 md:w-5 text-muted-foreground hidden md:block" />
              <h1 className="font-semibold text-foreground text-lg md:text-base">Manage Shares</h1>
            </div>
          </header>

          <main className="flex-1 overflow-auto p-2 md:p-6 bg-background">
          <div className="w-full">
            {loading ? (
              <div className="flex justify-center p-8 text-muted-foreground">Loading shares...</div>
            ) : shares.length === 0 ? (
              <div className="flex flex-col items-center justify-center p-12 text-muted-foreground border rounded-lg bg-card/50 mt-4 md:mt-0">
                <Share2 className="h-12 w-12 mb-4 opacity-20" />
                <p>You haven't shared any files yet.</p>
              </div>
            ) : (
              <div className="flex flex-col gap-4">
                {/* Desktop Column Headers */}
                <div className="hidden md:flex px-6 py-3 text-sm font-medium text-muted-foreground border-b bg-muted/30">
                  <div className="w-[140px]">Type</div>
                  <div className="flex-1">Path</div>
                  <div className="w-[80px] text-right">Links</div>
                  <div className="w-[40px]"></div>
                </div>

                <div className="flex flex-col space-y-2 md:space-y-3">
                  {Object.entries(groupedShares).map(([path, pathShares]) => {
                    const isExpanded = expandedPaths[path];
                    const firstShare = pathShares[0];

                    return (
                      <div key={path} className="border rounded-lg bg-card overflow-hidden flex flex-col shadow-sm transition-all">
                        <div
                          className="px-4 py-4 md:px-6 md:py-3 flex items-center justify-between cursor-pointer hover:bg-muted/50 transition-colors"
                          onClick={() => { togglePath(path); }}
                        >
                          {/* Mobile Layout: Type and Path Stacked */}
                          <div className="md:hidden flex flex-col gap-1.5 overflow-hidden mr-4 flex-1">
                            <div className="flex items-center gap-3 font-medium">
                              {firstShare.is_dir ? <Folder className="h-6 w-6 md:h-5 md:w-5 shrink-0 text-primary fill-primary/20" /> : <FileText className="h-6 w-6 md:h-5 md:w-5 shrink-0 text-gray-400" />}
                              <span className="text-base truncate" title={path}>{path}</span>
                            </div>
                            <div className="text-sm text-muted-foreground flex items-center ml-9">
                              <span className="bg-secondary/80 px-2.5 py-0.5 rounded-full text-xs font-medium">
                                {pathShares.length} link{pathShares.length !== 1 && "s"}
                              </span>
                            </div>
                          </div>

                          {/* Desktop Layout: Type and Path Separated */}
                          <div className="hidden md:flex items-center flex-1 overflow-hidden">
                            <div className="w-[140px] shrink-0 font-medium flex items-center gap-3">
                              {firstShare.is_dir ? (
                                <><Folder className="h-5 w-5 text-primary fill-primary/20" /> Folder</>
                              ) : (
                                <><FileText className="h-5 w-5 text-gray-400" /> File</>
                              )}
                            </div>
                            <div className="flex-1 font-medium truncate flex items-center text-[15px]" title={path}>
                              {path}
                            </div>
                            <div className="w-[80px] text-right pr-4">
                              <span className="text-xs text-muted-foreground bg-secondary/80 px-2.5 py-1 rounded-full font-medium">
                                {pathShares.length}
                              </span>
                            </div>
                          </div>

                          <div className="shrink-0 flex items-center justify-center h-8 w-8 rounded-full hover:bg-black/5 dark:hover:bg-white/10 transition-colors">
                            {isExpanded ? <ChevronUp className="h-5 w-5 text-muted-foreground" /> : <ChevronDown className="h-5 w-5 text-muted-foreground" />}
                          </div>
                        </div>

                        {/* Expanded Details Sub-list */}
                        {isExpanded && (
                          <div className="bg-muted/10 border-t flex flex-col divide-y divide-border/50">
                            {pathShares.map(share => {
                              const never = isNeverExpires(share.expires_at);
                              const isExpired = !never && new Date(share.expires_at) < new Date();
                              return (
                                <div key={share.id} className={`p-4 md:px-6 md:py-4 flex flex-col md:flex-row md:items-center gap-3 md:gap-6 ${share.blocked || isExpired ? "opacity-60 bg-muted/30" : "hover:bg-muted/20"}`}>
                                  <div className="flex flex-col gap-1.5 flex-1 min-w-0">
                                    <div className="flex items-center justify-between md:justify-start md:gap-4">
                                      <div className="text-[15px] font-medium truncate">
                                        {share.description || <span className="text-muted-foreground/70 italic text-sm">No description</span>}
                                      </div>
                                      <div className="md:hidden">
                                        {share.blocked ? (
                                          <span className="text-red-500 font-medium text-xs flex items-center gap-1 bg-red-500/10 px-2 py-0.5 rounded-sm">
                                            <Ban className="h-3 w-3" /> Blocked
                                          </span>
                                        ) : isExpired ? (
                                          <span className="text-muted-foreground font-medium text-xs bg-muted px-2 py-0.5 rounded-sm">Expired</span>
                                        ) : (
                                          <span className="text-green-500 font-medium text-xs flex items-center gap-1 bg-green-500/10 px-2 py-0.5 rounded-sm">
                                            <CheckCircle className="h-3 w-3" /> Active
                                          </span>
                                        )}
                                      </div>
                                    </div>
                                    
                                    <div className="flex flex-col md:flex-row md:items-center gap-1 md:gap-4 text-xs text-muted-foreground">
                                      <span className="font-mono bg-muted/50 px-1.5 py-0.5 rounded" title={`ID: ${share.id}`}>ID: {share.id.substring(0, 8)}...</span>
                                      <span className="flex items-center">
                                        {never ? "Never Expires" : <>Expires: {new Date(share.expires_at).toLocaleString()}</>}
                                        {isExpired && <span className="ml-1.5 text-red-500 font-semibold">(Expired)</span>}
                                      </span>
                                    </div>
                                  </div>

                                  <div className="hidden md:flex items-center justify-center w-[100px] shrink-0">
                                    {share.blocked ? (
                                      <span className="text-red-500 font-medium text-sm flex items-center gap-1.5">
                                        <Ban className="h-4 w-4" /> Blocked
                                      </span>
                                    ) : isExpired ? (
                                      <span className="text-muted-foreground font-medium text-sm flex items-center gap-1.5">
                                        Expired
                                      </span>
                                    ) : (
                                      <span className="text-green-500 font-medium text-sm flex items-center gap-1.5">
                                        <CheckCircle className="h-4 w-4" /> Active
                                      </span>
                                    )}
                                  </div>

                                  <div className="flex justify-end gap-2 pt-3 md:pt-0 border-t md:border-t-0 border-border/50 mt-1 md:mt-0 shrink-0">
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className={`h-8 ${share.blocked ? 'text-green-600 hover:text-green-700' : 'text-orange-600 hover:text-orange-700'}`}
                                      onClick={() => handleToggleBlock(share.id, share.blocked)}
                                    >
                                      {share.blocked ? "Unblock" : "Block"}
                                    </Button>
                                    <Button
                                      variant="destructive"
                                      size="icon"
                                      className="h-8 w-8"
                                      title="Delete Share"
                                      onClick={() => { openDeleteDialog(share.id); }}
                                    >
                                      <Trash2 className="h-4 w-4" />
                                    </Button>
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
            
            <ManageSharesDeleteDialog
              isOpen={isDeleteDialogOpen}
              onOpenChange={setIsDeleteDialogOpen}
              onConfirm={handleDelete}
              isLoading={isDeleting}
            />
          </div>
        </main>
        </div>
      </div>
    </DefaultLayout>
  );
}
