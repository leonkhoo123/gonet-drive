import axiosLayer from './axiosLayer';
import { unwrap, type ApiEnvelope } from './envelope';

export interface PinnedFolder {
  id: number;
  username: string;
  path: string;
  position: number;
  created_at: string;
}

export const getPinnedFolders = async (): Promise<PinnedFolder[]> => {
  const rs = await axiosLayer.get<ApiEnvelope<{ pins: PinnedFolder[] }>>('/user/pins');
  return unwrap<{ pins: PinnedFolder[] }>(rs).pins;
};

export const addPinnedFolder = async (path: string): Promise<void> => {
  await axiosLayer.post('/user/pin', { path });
};

export const removePinnedFolder = async (path: string): Promise<void> => {
  await axiosLayer.delete('/user/pin', { params: { path } });
};

export const reorderPinnedFolders = async (paths: string[]): Promise<void> => {
  await axiosLayer.put('/user/pins/reorder', { paths });
};
