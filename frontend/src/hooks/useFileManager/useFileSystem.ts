import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocation, useNavigate } from "react-router-dom";
import { fetchDirList, type ItemsResponse } from "@/api/api-file";
import { wsClient, type OperationMessage } from "@/api/wsClient";
import { usePreferences } from "@/context/PreferencesContext";
import { decodeUrlToPath } from "@/utils/utils";

export type SortField = 'name' | 'size' | 'modified';
export type SortOrder = 'asc' | 'desc';

export function useFileSystem(baseRoute = "/home") {
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
  const fetchIdRef = useRef<number>(0);
  const lastRefreshTimeRef = useRef<number>(0);
  const throttledTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const currentPathRef = useRef<string>("/");
  currentPathRef.current = currentPath;

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
    if (throttledTimerRef.current !== null) {
      clearTimeout(throttledTimerRef.current);
      throttledTimerRef.current = null;
    }
    lastRefreshTimeRef.current = Date.now();

    const currentFetchId = ++fetchIdRef.current;
    setIsLoading(true);
    setError(false);
    try {
      const [itemsrs] = await Promise.all([
        fetchDirList(currentPathRef.current, showHidden, sortField ?? undefined, sortField ? sortOrder : undefined),
        new Promise(resolve => setTimeout(resolve, 200))
      ]);
      if (currentFetchId === fetchIdRef.current) {
        setItems(itemsrs);
        if (itemsrs.share_root) {
          setShareRoot(itemsrs.share_root);
        }
        setIsLoading(false);
      }
    } catch (err: unknown) {
      if (currentFetchId === fetchIdRef.current) {
        const error = err as { message?: string; response?: { status?: number } };
        console.error("MyErr: ", err);
        console.error("err.message: ", error.message);
        console.error(" err.response.status: ", error.response?.status);
        if (error.response?.status === 401) {
          if (!baseRoute.includes("/share")) void navigate("/login");
        }
        setError(true);
        setIsLoading(false);
      }
    }
  }, [showHidden, sortField, sortOrder, navigate, baseRoute]);

  const THROTTLE_MS = 4000;

  const handleThrottledRefresh = useCallback((targetDir?: string) => {
    if (targetDir !== undefined && targetDir !== currentPathRef.current) return;

    const now = Date.now();
    const elapsed = now - lastRefreshTimeRef.current;

    if (elapsed >= THROTTLE_MS) {
      void handleRefresh();
    } else {
      throttledTimerRef.current ??= setTimeout(() => {
        throttledTimerRef.current = null;
        void handleRefresh();
      }, THROTTLE_MS - elapsed);
    }
  }, [handleRefresh]);

  useEffect(() => {
    const loadFiles = async () => {
      const currentFetchId = ++fetchIdRef.current;
      setIsLoading(true);
      setError(false);
      
      const rawPath = decodeURIComponent(location.pathname.replace(baseRoute, "")) || "/";
      const path = decodeUrlToPath(rawPath);
      
      // Clear items to show skeleton ONLY on directory change
      if (path !== prevPathRef.current) {
        setItems(undefined);
        setCurrentPath(path); // Immediately update currentPath so handleRefresh uses the new path
        prevPathRef.current = path;
      }

      try {
        const [itemsrs] = await Promise.all([
          fetchDirList(path, showHidden, sortField ?? undefined, sortField ? sortOrder : undefined),
          new Promise(resolve => setTimeout(resolve, 200))
        ]);
        if (currentFetchId === fetchIdRef.current) {
          setItems(itemsrs);
          setCurrentPath(itemsrs.path);
          if (itemsrs.share_root) {
            setShareRoot(itemsrs.share_root);
          }
          setIsLoading(false);
        }
      } catch (err: unknown) {
        if (currentFetchId === fetchIdRef.current) {
          const error = err as { message?: string; response?: { status?: number } };
          console.error("MyErr: ", err);
          console.error("err.message: ", error.message);
          console.error(" err.response.status: ", error.response?.status);
          if (error.response?.status === 401) {
            if (!baseRoute.includes("/share")) void navigate("/login");
          }
          setError(true);
          setIsLoading(false);
        }
      }
    };

    void loadFiles();
  }, [location, showHidden, sortField, sortOrder, navigate, baseRoute]);

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

  useEffect(() => {
    return () => {
      if (throttledTimerRef.current !== null) {
        clearTimeout(throttledTimerRef.current);
        throttledTimerRef.current = null;
      }
    };
  }, []);

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
    handleThrottledRefresh,
    sortField,
    setSortField,
    sortOrder,
    setSortOrder,
    handleSortChange
  };
}
