import axiosLayer from './axiosLayer';

export interface VerifyShareResponse {
  message: string;
  authority: 'view' | 'modify';
}

export interface CreateShareRequest {
  path: string;
  description: string;
  expires_in_hours: number;
  authority: 'view' | 'modify';
}

export interface ShareInfo {
  id: string;
  path: string;
  expires_at: string;
  authority: string;
}

export interface CreateShareResponse {
  message: string;
  share: ShareInfo;
  pin: string;
}

export const createShare = async (req: CreateShareRequest): Promise<CreateShareResponse> => {
  const rs = await axiosLayer.post("/user/share/create", req, {
    headers: { "Accept": "application/json", "Content-Type": "application/json" },
  });
  return rs.data as CreateShareResponse;
};

export const checkSharePermission = async (id: string): Promise<VerifyShareResponse> => {
  const rs = await axiosLayer.get(`/share/check-permission/${id}`, {
    headers: { "Accept": "application/json" },
  });
  return rs.data as VerifyShareResponse;
};

export const verifySharePin = async (id: string, pin: string): Promise<VerifyShareResponse> => {
  const rs = await axiosLayer.post("/share/verify", {
    id,
    pin
  }, {
    headers: { "Accept": "application/json" },
  });
  return rs.data as VerifyShareResponse;
};

export interface ShareItem {
  id: string;
  path: string;
  expires_at: string;
  blocked: boolean;
  authority: string;
  username: string;
  description: string;
  created_at: string;
  is_dir: boolean;
}

export const getShares = async (): Promise<ShareItem[]> => {
  const rs = await axiosLayer.get<{ shares: ShareItem[] }>("/user/share/get-shares");
  return rs.data.shares;
};

export const toggleShareBlock = async (id: string): Promise<any> => {
  const rs = await axiosLayer.put(`/user/share/${id}/toggle-block`);
  return rs.data;
};

export const deleteShare = async (id: string): Promise<any> => {
  const rs = await axiosLayer.delete(`/user/share/${id}`);
  return rs.data;
};