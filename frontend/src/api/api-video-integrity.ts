import axiosLayer from './axiosLayer';
import { unwrap, type ApiEnvelope } from './envelope';

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
  const response = await axiosLayer.post<ApiEnvelope<{ opId: string; status: string }>>('/user/admin/video-integrity/scan');
  return unwrap<{ opId: string; status: string }>(response);
};

export const stopScan = async (): Promise<{ opId: string; status: string }> => {
  const response = await axiosLayer.post<ApiEnvelope<{ opId: string; status: string }>>('/user/admin/video-integrity/scan', {
    action: 'stop',
  });
  return unwrap<{ opId: string; status: string }>(response);
};

export const getStatus = async (): Promise<VideoIntegrityStatus> => {
  const response = await axiosLayer.get<ApiEnvelope<VideoIntegrityStatus>>('/user/admin/video-integrity/status');
  return unwrap<VideoIntegrityStatus>(response);
};

export const getList = async (): Promise<VideoIntegrityListResponse> => {
  const response = await axiosLayer.get<ApiEnvelope<VideoIntegrityListResponse>>('/user/admin/video-integrity/list');
  return unwrap<VideoIntegrityListResponse>(response);
};
