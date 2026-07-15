import axiosInstance from "./axiosLayer";
import { unwrap, type ApiEnvelope } from "./envelope";

export interface AudioBookItem {
  name: string;
  total_length: number;
  size: number;
  progress_time: number;
  last_modified: string;
  mediaType: string;
  url?: string;
}

export interface AudioPath {
  id: number;
  path: string;
  username: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export const getAudioBooks = async (): Promise<AudioBookItem[]> => {
  const response = await axiosInstance.get<ApiEnvelope<{ audiobooks: AudioBookItem[] }>>("/user/audiobook/list");
  return unwrap<{ audiobooks: AudioBookItem[] }>(response).audiobooks;
};

export const getAudioPaths = async (): Promise<AudioPath[]> => {
  const response = await axiosInstance.get<ApiEnvelope<{ paths: AudioPath[] }>>("/user/audiobook/paths");
  return unwrap<{ paths: AudioPath[] }>(response).paths;
};

export const addAudioPath = async (path: string, is_enabled = true): Promise<void> => {
  await axiosInstance.post("/user/audiobook/path", { path, is_enabled });
};

export const updateAudioPath = async (id: number, is_enabled: boolean): Promise<void> => {
  await axiosInstance.put(`/audiobook/path/${String(id)}`, { is_enabled });
};

export const deleteAudioPath = async (id: number): Promise<void> => {
  await axiosInstance.delete(`/audiobook/path/${String(id)}`);
};

export const reportAudioBookProgress = async (audiobook_name: string, progress_time: number): Promise<void> => {
  await axiosInstance.post("/user/audiobook/progress", { audiobook_name, progress_time });
};

export const getAudioBookFileList = async (path = "/.audio_book"): Promise<{ path: string, items: AudioBookItem[] }> => {
  const response = await axiosInstance.get<ApiEnvelope<{ path: string, items: AudioBookItem[] }>>("/user/audiobook/filelist", { params: { path } });
  return unwrap<{ path: string, items: AudioBookItem[] }>(response);
};

export const getAudioBookStreamUrl = (name: string): string => {
  return `${window.location.protocol}//${window.location.host}/api/user/audiobook/stream/${encodeURIComponent(name)}`;
};
