import { useState, useEffect, useCallback, useMemo } from 'react';
import { toast } from 'sonner';
import {
  type PinnedFolder,
  getPinnedFolders,
  addPinnedFolder,
  removePinnedFolder,
  reorderPinnedFolders,
} from '@/api/api-pinned';

const MAX_PINS = 10;

export function usePinnedFolders() {
  const [pinnedFolders, setPinnedFolders] = useState<PinnedFolder[]>([]);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);

  const fetchPins = useCallback(async () => {
    try {
      const pins = await getPinnedFolders();
      setPinnedFolders(pins);
    } catch {
      // silently fail — sidebar just shows no pins
    }
    setIsLoaded(true);
  }, []);

  useEffect(() => {
    void fetchPins();
  }, [fetchPins]);

  const pinnedPaths = useMemo(() => new Set(pinnedFolders.map((f) => f.path)), [pinnedFolders]);

  const pin = useCallback(async (path: string) => {
    if (pinnedPaths.has(path)) return;
    if (pinnedFolders.length >= MAX_PINS) {
      toast.error(`Maximum ${String(MAX_PINS)} pinned folders allowed`);
      return;
    }
    try {
      await addPinnedFolder(path);
      await fetchPins();
    } catch {
      toast.error('Failed to pin folder');
    }
  }, [pinnedPaths, pinnedFolders.length, fetchPins]);

  const unpin = useCallback(async (path: string) => {
    try {
      await removePinnedFolder(path);
      setPinnedFolders((prev) => prev.filter((f) => f.path !== path));
    } catch {
      toast.error('Failed to unpin folder');
    }
  }, []);

  const reorder = useCallback(async (paths: string[]) => {
    const updated = paths.map((path, i) => {
      const existing = pinnedFolders.find((f) => f.path === path);
      return existing ? { ...existing, position: i } : null;
    }).filter(Boolean) as PinnedFolder[];
    setPinnedFolders(updated);
    try {
      await reorderPinnedFolders(paths);
    } catch {
      toast.error('Failed to reorder pinned folders');
      void fetchPins();
    }
  }, [pinnedFolders, fetchPins]);

  const toggleEditMode = useCallback(() => {
    setIsEditMode((prev) => !prev);
  }, []);

  return {
    pinnedFolders,
    isEditMode,
    isLoaded,
    pinnedPaths,
    fetchPins,
    pin,
    unpin,
    reorder,
    toggleEditMode,
  };
}
