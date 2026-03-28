import { useState } from 'react';
import { toast } from "sonner";
import { postDisqualified, renameFileMoveToDone } from "@/api/api-video";
import { deleteTempRotate, type FileInterface } from "@/api/api-file";

export function useVideoOperations({
  currentPath,
  handleRefresh,
  setIsLoading,
  setError
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

  const removeRotateTemp = async () => {
    console.log("Removing temp_rotate");
    try {
      setIsLoading(true);
      await deleteTempRotate(currentPath);
      await handleRefresh();
    } catch (error: any) {
      setError(true);
      toast.error("Failed to Clean Up");
      console.log("Failed to remove rotate_temp", error);
    } finally {
      setIsLoading(false);
    }
  };

  return { selectedVideo, setSelectedVideo, handlePlayerClose, removeRotateTemp };
}
