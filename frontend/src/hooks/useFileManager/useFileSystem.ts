import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocation, useNavigate } from "react-router-dom";
import { fetchDirList, type ItemsResponse } from "@/api/api-file";
import { wsClient, type OperationMessage } from "@/api/wsClient";
import { usePreferences } from "@/context/PreferencesContext";
import { decodeUrlToPath } from "@/utils/utils";

export type SortField = 'name' | 'size' | 'modified';
export type SortOrder = 'asc' | 'desc';

export function useFileSystem(baseRoute: string = "/home") {
  const location = useLocation();
  const navigate = useNavigate();
  const { showHidden } = usePreferences();
  
  const [items, setItems] = useState<ItemsResponse>();
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<boolean>(false);
  const [currentPath, setCurrentPath] = useState<string>("/");
  const [shareRoot, setShareRoot] = useState<string>("");

  const [sortField, setSortField] = useState<SortField | null>(null);
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');

  const prevPathRef = useRef<string>("/");

  const handleSortChange = useCallback((field: SortField) => {
    if (sortField === field) {
      if (sortOrder === 'asc') {
        setSortOrder('desc');
      } else {
        setSortField(null);
        setSortOrder('asc');
      }
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  }, [sortField, sortOrder]);

  const handleRefresh = useCallback(async () => {
    setIsLoading(true);
    setError(false);
    try {
      const [itemsrs] = await Promise.all([
        fetchDirList(currentPath, showHidden, sortField ?? undefined, sortField ? sortOrder : undefined),
        new Promise(resolve => setTimeout(resolve, 200))
      ]);
      setItems(itemsrs);
      if (itemsrs.share_root) {
        setShareRoot(itemsrs.share_root);
      }
    } catch (err: any) {
      console.error("MyErr: ", err);
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      console.error("err.message: ", err.message);
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      console.error(" err.response.status: ", err?.response?.status);
      // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
      if (err?.response?.status === 401) {
        if (!baseRoute.includes("/share")) void navigate("/login");
      }
      setError(true);
    } finally {
      setIsLoading(false);
    }
  }, [currentPath, showHidden, sortField, sortOrder, navigate, baseRoute]);

  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (!location) return;

    const loadFiles = async () => {
      setIsLoading(true);
      setError(false);
      
      const rawPath = decodeURIComponent(location.pathname.replace(baseRoute, "")) || "/";
      const path = decodeUrlToPath(rawPath);
      
      // Clear items to show skeleton ONLY on directory change
      if (path !== prevPathRef.current) {
        setItems(undefined);
        prevPathRef.current = path;
      }

      try {
        const [itemsrs] = await Promise.all([
          fetchDirList(path, showHidden, sortField ?? undefined, sortField ? sortOrder : undefined),
          new Promise(resolve => setTimeout(resolve, 200))
        ]);
        setItems(itemsrs);
        setCurrentPath(itemsrs.path);
        if (itemsrs.share_root) {
          setShareRoot(itemsrs.share_root);
        }
      } catch (err: any) {
        console.error("MyErr: ", err);
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        console.error("err.message: ", err.message);
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        console.error(" err.response.status: ", err?.response?.status);
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        if (err?.response?.status === 401) {
          if (!baseRoute.includes("/share")) void navigate("/login");
        }
        setError(true);
      } finally {
        setIsLoading(false);
      }
    };

    void loadFiles();
  }, [location, showHidden, sortField, sortOrder, navigate]);

  useEffect(() => {
    const unsubscribe = wsClient.subscribe((msg: OperationMessage) => {
      if (msg.opStatus === 'completed') {
        if (
          msg.destDir === currentPath || 
          msg.opType === 'delete_permanent' || 
          msg.opType === 'delete'
        ) {
          void handleRefresh();
        }
      }
    });

    return () => {
      unsubscribe();
    };
  }, [currentPath, handleRefresh]);

  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        void handleRefresh();
      }
    };

    const handleFocus = () => {
      void handleRefresh();
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("focus", handleFocus);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("focus", handleFocus);
    };
  }, [handleRefresh]);

  return {
    items,
    setItems,
    isLoading,
    setIsLoading,
    error,
    setError,
    currentPath,
    shareRoot,
    handleRefresh,
    sortField,
    sortOrder,
    handleSortChange
  };
}
