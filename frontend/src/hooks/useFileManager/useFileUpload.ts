import { useState, useCallback } from 'react';
import { toast } from "sonner";
import axios from 'axios';
import { uploadFile, checkUploadDuplicates, type UploadProgressEvent, type DuplicateItem } from "@/api/api-file";
import { useOperationProgress } from "@/context/OperationProgressContext";
import { formatBytes } from "@/utils/utils";

export function useFileUpload(handleRefresh: () => Promise<void>, uploadChunkSize?: number) {
  const { addOrUpdateOperation } = useOperationProgress();
  const [isUploadDuplicateCheckDialogOpen, setIsUploadDuplicateCheckDialogOpen] = useState(false);
  const [isUploadDuplicateChecking, setIsUploadDuplicateChecking] = useState(false);
  const [uploadDuplicateItems, setUploadDuplicateItems] = useState<DuplicateItem[]>([]);
  const [pendingUploads, setPendingUploads] = useState<{ files: File[], targetPath: string } | null>(null);

  const executeUpload = useCallback(async (directFiles?: File[], directTargetPath?: string) => {
    const filesToUpload = directFiles ?? pendingUploads?.files;
    const target = directTargetPath ?? pendingUploads?.targetPath;

    if (!filesToUpload || !target) return;
    
    setIsUploadDuplicateCheckDialogOpen(false);
    setPendingUploads(null);

    const newUploads = filesToUpload.map(file => ({
      id: "upload-" + Math.random().toString(36).substring(7),
      name: file.name,
    }));

    // Initialize all uploads in context
    newUploads.forEach(upload => {
      addOrUpdateOperation({
        opId: upload.id,
        opType: 'upload',
        opName: `Uploading ${upload.name}`,
        opStatus: 'queued',
        destDir: target,
        opPercentage: 0,
      });
    });

    for (let i = 0; i < filesToUpload.length; i++) {
      const file = filesToUpload[i];
      const upload = newUploads[i];

      addOrUpdateOperation({
        opId: upload.id,
        opType: 'upload',
        opName: `Uploading ${upload.name}`,
        opStatus: 'in-progress',
        destDir: target,
        opPercentage: 0,
      });

      try {
        await uploadFile(target, file, (progressEvent: UploadProgressEvent) => {
          if (progressEvent.total) {
            const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
            let speedText = progressEvent.rate ? `${formatBytes(progressEvent.rate)}/s` : '';
            if (progressEvent.estimated) {
                const est = progressEvent.estimated;
                const minutes = Math.floor(est / 60);
                const seconds = Math.floor(est % 60);
                const timeStr = minutes > 0 ? `${String(minutes)}m ${String(seconds)}s` : `${String(seconds)}s`;
                speedText += speedText ? ` • ${timeStr}` : timeStr;
            }
            const opSpeed = speedText || undefined;
            addOrUpdateOperation({
              opId: upload.id,
              opType: 'upload',
              opName: `Uploading ${upload.name}`,
              opStatus: 'in-progress',
              destDir: target,
              opPercentage: progress,
              opSpeed,
            });
          }
        }, upload.id, uploadChunkSize);
        
        addOrUpdateOperation({
          opId: upload.id,
          opType: 'upload',
          opName: `Uploaded ${upload.name}`,
          opStatus: 'completed',
          destDir: target,
          opPercentage: 100,
        });
      } catch (error: unknown) {
        console.error("Upload error:", error);
        let errorMessage = error instanceof Error ? error.message : 'Upload failed';
        
        if (axios.isAxiosError(error) && error.response?.data) {
          const errData = error.response.data as { error?: string };
          if (errData.error) {
            errorMessage = errData.error;
          }
        }
        
        const isCancelled = errorMessage === "Upload cancelled";
        
        addOrUpdateOperation({
          opId: upload.id,
          opType: 'upload',
          opName: isCancelled ? `Cancelled upload for ${upload.name}` : `Failed to upload ${upload.name}`,
          opStatus: isCancelled ? 'aborted' : 'error',
          destDir: target,
          error: isCancelled ? undefined : errorMessage,
        });
        
        if (!isCancelled) {
            toast.error(`Failed to upload ${file.name}`);
        }
      }
    }
    
    // Refresh after all uploads finish
    if (filesToUpload.length > 0) {
      void handleRefresh();
    }
  }, [pendingUploads, addOrUpdateOperation, handleRefresh, uploadChunkSize]);

  const handleUploadFiles = useCallback(async (files: File[], targetPath: string) => {
    if (files.length === 0) return;

    try {
      setIsUploadDuplicateCheckDialogOpen(true);
      setIsUploadDuplicateChecking(true);
      setUploadDuplicateItems([]);
      setPendingUploads({ files, targetPath });

      const fileDetails = files.map(f => {
        const customPath = ((f as unknown) as { customPath?: string }).customPath ?? f.webkitRelativePath;
        return {
          path: customPath || f.name,
          size: f.size,
          modifiedAt: new Date(f.lastModified).toISOString()
        };
      });

      const startTime = Date.now();
      const res = await checkUploadDuplicates(fileDetails, targetPath);
      
      const elapsedTime = Date.now() - startTime;
      if (elapsedTime < 300) {
        await new Promise(resolve => setTimeout(resolve, 300 - elapsedTime));
      }

      if (res.hasDuplicates) {
        setIsUploadDuplicateChecking(false);
        setUploadDuplicateItems(res.duplicates);
      } else {
        setIsUploadDuplicateCheckDialogOpen(false);
        void executeUpload(files, targetPath);
        setTimeout(() => { setIsUploadDuplicateChecking(false); }, 300);
      }
    } catch (error) {
      console.error("Check upload duplicates failed:", error);
      toast.error("Failed to check for duplicates before upload");
      setIsUploadDuplicateChecking(false);
      setIsUploadDuplicateCheckDialogOpen(false);
      setPendingUploads(null);
    }
  }, [executeUpload]);

  return { handleUploadFiles, executeUpload, isUploadDuplicateCheckDialogOpen, setIsUploadDuplicateCheckDialogOpen, isUploadDuplicateChecking, uploadDuplicateItems, setPendingUploads };
}
