import { useState } from 'react';
import { postDisqualified, renameFileMoveToDone } from "@/api/api-video";
import { type FileInterface } from "@/api/api-file";

export function useVideoOperations({
  handleRefresh
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
        // The server atomically moves the video into done/tmp before it
        // responds, so refresh now to make it disappear from the browse list
        // while the rotate/embed job runs in the background.
        await renameFileMoveToDone(oriPath, newName, rotation);
        await handleRefresh();
      }
    } catch (error) {
      console.error("Failed to move or rename file:", error);
    }
  };

  return { selectedVideo, setSelectedVideo, handlePlayerClose };
}
