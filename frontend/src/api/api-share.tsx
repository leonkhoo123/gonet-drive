import axiosLayer from './axiosLayer';
import { unwrap, type ApiEnvelope } from './envelope';

// isNeverExpires checks if the expires_at string is the sentinel "never" value (year 9999).
export const isNeverExpires = (expiresAt: string): boolean => {
  return new Date(expiresAt).getFullYear() >= 9999;
};

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
  const rs = await axiosLayer.post<ApiEnvelope<CreateShareResponse>>("/user/share/create", req, {
    headers: { "Accept": "application/json", "Content-Type": "application/json" },
  });
  return unwrap<CreateShareResponse>(rs);
};

export const checkSharePermission = async (id: string): Promise<VerifyShareResponse> => {
  const rs = await axiosLayer.get<ApiEnvelope<VerifyShareResponse>>(`/share/check-permission/${id}`, {
    headers: { "Accept": "application/json" },
  });
  return unwrap<VerifyShareResponse>(rs);
};

export const verifySharePin = async (id: string, pin: string): Promise<VerifyShareResponse> => {
  const rs = await axiosLayer.post<ApiEnvelope<VerifyShareResponse>>("/share/verify", {
    id,
    pin
  }, {
    headers: { "Accept": "application/json" },
  });
  return unwrap<VerifyShareResponse>(rs);
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
  const rs = await axiosLayer.get<ApiEnvelope<{ shares: ShareItem[] }>>("/user/share/get-shares");
  return unwrap<{ shares: ShareItem[] }>(rs).shares;
};

export const toggleShareBlock = async (id: string): Promise<unknown> => {
  const rs = await axiosLayer.put<ApiEnvelope<unknown>>(`/user/share/${id}/toggle-block`);
  return unwrap<unknown>(rs);
};

export const deleteShare = async (id: string): Promise<unknown> => {
  const rs = await axiosLayer.delete<ApiEnvelope<unknown>>(`/user/share/${id}`);
  return unwrap<unknown>(rs);
};