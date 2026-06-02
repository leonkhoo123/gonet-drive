import axiosLayer from './axiosLayer';

export interface VideoIntegrityEntry {
  hash: string;
  file_path: string;
  relative_path: string;
  issue_type: string;
  mime_codec_string: string;
  detected_at: string;
  last_checked_at: string;
}

export interface VideoIntegrityStatus {
  corrupt_count: number;
  scan_running: boolean;
  last_scan?: string;
}

export interface VideoIntegrityListResponse {
  total: number;
  entries: VideoIntegrityEntry[];
}

export const startScan = async (): Promise<{ opId: string; status: string }> => {
  const response = await axiosLayer.post<{ opId: string; status: string }>(
    '/user/admin/video-integrity/scan'
  );
  return response.data;
};

export const stopScan = async (): Promise<{ opId: string; status: string }> => {
  const response = await axiosLayer.post<{ opId: string; status: string }>(
    '/user/admin/video-integrity/scan',
    { action: 'stop' }
  );
  return response.data;
};

export const getStatus = async (): Promise<VideoIntegrityStatus> => {
  const response = await axiosLayer.get<VideoIntegrityStatus>(
    '/user/admin/video-integrity/status'
  );
  return response.data;
};

export const getList = async (): Promise<VideoIntegrityListResponse> => {
  const response = await axiosLayer.get<VideoIntegrityListResponse>(
    '/user/admin/video-integrity/list'
  );
  return response.data;
};
