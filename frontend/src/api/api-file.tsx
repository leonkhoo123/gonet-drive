import axiosLayer from './axiosLayer';   // axios instance WITHOUT token
import axios from 'axios';
import { generateOpId } from "../utils/id";
import { getConfig } from '../config';

export let isShareMode = false;
export const setShareMode = (val: boolean) => { isShareMode = val; };

export interface ItemsResponse {
  items: FileInterface[];
  path: string;
  share_root?: string;
  file_count?: number;
  folder_count?: number;
  count?: number;
  is_single_file?: boolean;
  storage?: StorageUsageResponse;
}

export interface FileInterface {
  modified: string;
  name: string;
  size: number;
  type: 'file' | 'dir';
  url: string;
  media_type?: string;
  path: string;
  isShared?: boolean;
}


export interface HealthResponse {
  service_name?: string;
  upload_chunk_size?: number;
  video_mode?: string;
  [key: string]: any;
}

export const checkHealth = async (): Promise<HealthResponse | null> => {
  try {
    const rs = await axiosLayer.get<HealthResponse>("/health");
    if (rs.status === 200) {
      return rs.data;
    }
    return null;
  } catch {
    return null;
  }
};

export const fetchDirList = async (path = "/", showHidden = false, sort?: string, order?: string): Promise<ItemsResponse> => {
  // Clean up path formatting (avoid duplicate slashes)
  const cleanPath = path.trim() === "" ? "/" : path;

  const params: Record<string, any> = { path: cleanPath, showHidden };
  if (sort) params.sort = sort;
  if (order) params.order = order;

  const endpoint = isShareMode ? "/share/file/list" : "/user/files/file-list";
  const rs = await axiosLayer.get(endpoint, {
    params,
    headers: { "Accept": "application/json" },
  });

  return rs.data as ItemsResponse;
};

export const copyFiles = async (sources: string[], destDir: string, opId: string = generateOpId()) => {
  const endpoint = isShareMode ? "/share/file/copy" : "/user/files/copy";
  const rs = await axiosLayer.post(endpoint, {
    sources,
    destDir,
    opId
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

export const moveFiles = async (sources: string[], destDir: string, opId: string = generateOpId()) => {
  const endpoint = isShareMode ? "/share/file/move" : "/user/files/move";
  const rs = await axiosLayer.post(endpoint, {
    sources,
    destDir,
    opId
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

export const deleteFiles = async (sources: string[], opId: string = generateOpId()) => {
  const endpoint = isShareMode ? "/share/file/delete" : "/user/files/delete";
  const rs = await axiosLayer.post(endpoint, {
    sources,
    opId
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

export const deletePermanentFiles = async (sources: string[], opId: string = generateOpId()) => {
  const endpoint = isShareMode ? "/share/file/delete-permanent" : "/user/files/delete-permanent";
  const rs = await axiosLayer.post(endpoint, {
    sources,
    opId
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

export const renameFile = async (source: string, newName: string, opId: string = generateOpId()) => {
  const endpoint = isShareMode ? "/share/file/rename" : "/user/files/rename";
  const rs = await axiosLayer.post(endpoint, {
    source,
    newName,
    opId
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

export const createFolder = async (path: string, folderName: string, opId: string = generateOpId()) => {
  const endpoint = isShareMode ? "/share/file/create-folder" : "/user/files/create-folder";
  const rs = await axiosLayer.post(endpoint, {
    dir: path,
    folderName,
    opId
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

export interface FileDetail {
  name: string;
  isDir: boolean;
  size: number;
  modifiedAt: string;
}

export interface DuplicateItem {
  source: FileDetail;
  target: FileDetail;
}

export interface CheckDuplicatesResponse {
  hasDuplicates: boolean;
  duplicates: DuplicateItem[];
}

export const checkDuplicates = async (sources: string[], destDir: string): Promise<CheckDuplicatesResponse> => {
  if (isShareMode) return { hasDuplicates: false, duplicates: [] };
  const rs = await axiosLayer.post("/user/files/check-duplicates", {
    sources,
    destDir
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data as CheckDuplicatesResponse;
};

export interface UploadFileDetailReq {
  path: string;
  size: number;
  modifiedAt: string;
}

export const checkUploadDuplicates = async (files: UploadFileDetailReq[], destDir: string): Promise<CheckDuplicatesResponse> => {
  if (isShareMode) return { hasDuplicates: false, duplicates: [] };
  const rs = await axiosLayer.post("/user/files/check-upload-duplicates", {
    files,
    destDir
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data as CheckDuplicatesResponse;
};

export interface PropertiesContains {
  files: number;
  folders: number;
}

export interface PropertiesResponse {
  type: "file" | "directory" | "multiple";
  name: string | null;
  location: string;
  totalSizeBytes: number;
  contains: PropertiesContains;
  createdAt: string | null;
  modifiedAt: string | null;
  accessedAt: string | null;
}

export const getFileProperties = async (sources: string[]): Promise<PropertiesResponse> => {
  const endpoint = isShareMode ? "/share/file/properties" : "/user/files/properties";
  const rs = await axiosLayer.post(endpoint, {
    sources
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data as PropertiesResponse;
};

export const cancelOperation = async (opId: string, cancel = true) => {
  const rs = await axiosLayer.post("/user/files/cancel", {
    opId,
    cancel
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data;
};

import * as CRC32 from 'crc-32';
import { toast } from 'sonner';

export const downloadFiles = (sources: string[]) => {
  if (sources.length === 0) return;
  const baseUrl = getConfig().apiBaseUrl;
  const params = new URLSearchParams();
  sources.forEach(src => { params.append('source', src); });
  
  const endpoint = isShareMode ? "/share/file/download" : "/user/files/download";
  if (isShareMode) {
    const pathParts = window.location.pathname.split('/');
    const shareId = pathParts[1] === 'share' ? pathParts[2] : null;
    if (shareId) params.append('share_id', shareId);
  }
  const url = `${baseUrl}${endpoint}?${params.toString()}`;

  // Check if device is iOS (including iPadOS)
  const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent) || 
                (navigator.userAgent.includes("Mac") && "ontouchend" in document);

  if (isIOS) {
    // In iOS PWAs, auth cookies are not shared with window.open (Safari View Controller),
    // and standard <a> tag downloads are buffered silently into memory before showing a Quick Look preview.
    // The industry practice here is to inform the user of this OS limitation so they know it's working.
    toast.info("Download started in background", {
      description: "Please wait. iOS will pop up a save menu when the file is fully ready.",
      duration: 8000,
    });
  } else {
    toast.success("Download started", {
      description: "Check your device's notification center or downloads folder.",
      duration: 5000,
    });
  }
  
  // Create an invisible anchor tag to trigger download
  const a = document.createElement('a');
  a.style.display = 'none';
  a.href = url;
  
  // Set explicit download filename if only one file is selected to prevent mobile browsers from appending .txt
  if (sources.length === 1) {
    const filename = sources[0].split('/').pop() ?? '';
    a.setAttribute('download', filename);
  } else {
    a.setAttribute('download', '');
  }
  
  document.body.appendChild(a);
  a.click();
  
  // Clean up
  setTimeout(() => {
    document.body.removeChild(a);
  }, 100);
};

export interface UploadProgressEvent {
  loaded: number;
  total: number;
  progress: number;
  bytes: number;
  rate?: number;
  estimated?: number;
  upload: boolean;
}

export const uploadControllers = new Map<string, AbortController>();
export const cancelledUploads = new Set<string>();

export const uploadFile = async (
  path: string,
  file: File,
  onProgress?: (progressEvent: UploadProgressEvent) => void,
  opId?: string,
  chunkSize?: number
// eslint-disable-next-line @typescript-eslint/no-explicit-any
): Promise<any> => {
  let CHUNK_SIZE = chunkSize && chunkSize > 0 ? chunkSize : 5 * 1024 * 1024; // Use provided size or default to 5MB
  let totalChunks = Math.ceil(file.size / CHUNK_SIZE) || 1; // at least 1 chunk for empty files

  if (file.size >= 3 && totalChunks < 3) {
    CHUNK_SIZE = Math.floor(file.size / 3);
    totalChunks = Math.ceil(file.size / CHUNK_SIZE);
  }

  let loadedBytes = 0;
  
  // Extract subdirectory if it's a folder upload (webkitRelativePath contains the full path including filename)
  let destination = path;
  let filename = file.name;
  
  const customPath = ((file as unknown) as { customPath?: string }).customPath ?? file.webkitRelativePath;
  if (customPath) {
    const parts = customPath.split('/');
    if (parts.length > 1) {
      filename = parts.pop() ?? file.name; // last part is filename
      // Join remaining parts to form the relative path, and append to base destination
      const relativeFolder = parts.join('/');
      destination = path.endsWith('/') 
        ? `${path}${relativeFolder}` 
        : `${path}/${relativeFolder}`;
    }
  }

  const identifier = opId ?? btoa(encodeURIComponent(`${filename}-${String(file.size)}-${String(file.lastModified)}-${destination}`)).replace(/[/+=]/g, '_');

  if (cancelledUploads.has(identifier)) {
    cancelledUploads.delete(identifier);
    throw new Error("Upload cancelled");
  }

  const controller = new AbortController();
  uploadControllers.set(identifier, controller);

  let lastResponse = null;
  const startTime = Date.now();

  try {
    for (let chunkNumber = 1; chunkNumber <= totalChunks; chunkNumber++) {
      if (controller.signal.aborted) {
        throw new Error("Upload cancelled");
      }
      
      const start = (chunkNumber - 1) * CHUNK_SIZE;
      const end = Math.min(start + CHUNK_SIZE, file.size);
      const chunkBlob = file.slice(start, end);
      
      // Calculate CRC32 checksum for the chunk
      const arrayBuffer = await chunkBlob.arrayBuffer();
      const buffer = new Uint8Array(arrayBuffer);
      const checksumNum = CRC32.buf(buffer);
      const checksum = (checksumNum >>> 0).toString(16).padStart(8, '0');

      let status = 'uploading';
      if (chunkNumber === 1 && totalChunks === 1) status = 'end';
      else if (chunkNumber === 1) status = 'start';
      else if (chunkNumber === totalChunks) status = 'end';

      const formData = new FormData();
      formData.append("identifier", identifier);
      formData.append("filename", filename);
      formData.append("destination", destination);
      formData.append("chunkNumber", chunkNumber.toString());
      formData.append("totalChunks", totalChunks.toString());
      formData.append("status", status);
      formData.append("checksum", checksum);
      formData.append("chunk", chunkBlob, filename);

      const endpoint = isShareMode ? "/share/file/upload-chunk" : "/user/files/upload-chunk";
      const rs = await axiosLayer.post(endpoint, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
        signal: controller.signal,
        onUploadProgress: (progressEvent) => {
          if (onProgress && progressEvent.loaded) {
            // Calculate overall progress across chunks
            const overallLoaded = loadedBytes + progressEvent.loaded;
            
            const elapsedSeconds = Math.max((Date.now() - startTime) / 1000, 0.1);
            const calculatedRate = overallLoaded / elapsedSeconds;
            const calculatedEstimated = file.size > overallLoaded 
              ? (file.size - overallLoaded) / calculatedRate 
              : 0;

            onProgress({
              loaded: overallLoaded,
              total: file.size,
              progress: file.size > 0 ? overallLoaded / file.size : 1,
              bytes: progressEvent.bytes,
              rate: calculatedRate,
              estimated: calculatedEstimated,
              upload: true
            });
          }
        },
      });

      loadedBytes += chunkBlob.size;
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      lastResponse = rs.data;
    }
  } catch (error: unknown) {
    if (axios.isCancel(error) || (error instanceof Error && error.message === "Upload cancelled")) {
      // Send cancel request without payload
      const formData = new FormData();
      formData.append("identifier", identifier);
      formData.append("filename", filename);
      formData.append("destination", destination);
      formData.append("status", "cancel");

      try {
        const endpoint = isShareMode ? "/share/file/upload-chunk" : "/user/files/upload-chunk";
        await axiosLayer.post(endpoint, formData, {
          headers: {
            "Content-Type": "multipart/form-data",
          },
        });
      } catch (cancelError) {
        console.error("Failed to send cancel status", cancelError);
      }
      throw new Error("Upload cancelled");
    }
    throw error;
  } finally {
    uploadControllers.delete(identifier);
  }

  return lastResponse;
};


export interface StorageUsageResponse {
  used: number;
  limit: number;
  left: number;
}
