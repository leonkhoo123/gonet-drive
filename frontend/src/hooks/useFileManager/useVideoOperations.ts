import { useState } from 'react';
import { postDisqualified, renameFileMoveToDone } from "@/api/api-video";
import { type FileInterface } from "@/api/api-file";

export function useVideoOperations({
  handleRefresh,
  setIsLoading
}: {
  currentPath: string;
  handleRefresh: () => Promise<void>;
  setIsLoading: (loading: boolean) => void;
  setError: (error: boolean) => void;
}) {
  const [selectedVideo, setSelectedVideo] = useState<FileInterface | null>(null);

  const handlePlayerClose = async (isDisqualified: boolean, oriPath: string, isNewName: boolean, newName: string, rotation: number): Promise<void> => {
    setSelectedVideo(null);
    try {
      if (isDisqualified) {
        await postDisqualified(oriPath);
        await handleRefresh();
      } else if (isNewName) {
        setIsLoading(true);
        await renameFileMoveToDone(oriPath, newName, rotation);
        setIsLoading(false);
        await handleRefresh();
      }
    } catch (error) {
      console.error("Failed to move or rename file:", error);
    }
  };

  return { selectedVideo, setSelectedVideo, handlePlayerClose };
}
