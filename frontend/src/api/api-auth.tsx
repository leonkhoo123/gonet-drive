import axiosLayer from "./axiosLayer";
import fpPromise from '@fingerprintjs/fingerprintjs';
import { unwrap, type ApiEnvelope } from "./envelope";

// Initialize the agent at application startup.
const getFingerprint = async (): Promise<string> => {
  const fp = await fpPromise.load();
  const result = await fp.get();
  return result.visitorId;
};

export interface LoginResponse {
  auth_status: "logged_in" | "mfa_required";
  mfa_setup_required?: boolean;
}

export const login = async (username: string, password: string): Promise<LoginResponse> => {
    const device_id = await getFingerprint();
    const res = await axiosLayer.post<ApiEnvelope<LoginResponse>>(
      "/login",
      { username, password, device_id },
      { 
        headers: { "Content-Type": "application/json" },
      }
    );
    return unwrap<LoginResponse>(res);
};

export const verifyMfa = async (code: string): Promise<LoginResponse> => {
    const device_id = await getFingerprint();
    const res = await axiosLayer.post<ApiEnvelope<LoginResponse>>(
      "/mfa/verify",
      { code, device_id },
      { 
        headers: { "Content-Type": "application/json" },
      }
    );
    return unwrap<LoginResponse>(res);
};

export interface MfaSetupResponse {
  secret: string;
  url: string;
}

export const setupMfa = async (): Promise<MfaSetupResponse> => {
    const res = await axiosLayer.post<ApiEnvelope<MfaSetupResponse>>("/user/mfa/setup");
    return unwrap<MfaSetupResponse>(res);
};

export interface MfaEnableResponse {
  recovery_codes: string[];
}

export const enableMfa = async (code: string): Promise<MfaEnableResponse> => {
    const res = await axiosLayer.post<ApiEnvelope<MfaEnableResponse>>(
      "/user/mfa/confirm",
      { code },
      { 
        headers: { "Content-Type": "application/json" },
      }
    );
    return unwrap<MfaEnableResponse>(res);
};

export const verifyMfaRecovery = async (code: string): Promise<LoginResponse> => {
    const device_id = await getFingerprint();
    const res = await axiosLayer.post<ApiEnvelope<LoginResponse>>(
      "/mfa/recovery",
      { recovery_code: code, device_id },
      { 
        headers: { "Content-Type": "application/json" },
      }
    );
    return unwrap<LoginResponse>(res);
};

export const getMe = async (): Promise<{ username: string; role: string }> => {
  const res = await axiosLayer.get<ApiEnvelope<{ username: string; role: string }>>("/user/me");
  return unwrap<{ username: string; role: string }>(res);
};

export const checkAuthStatus = async (): Promise<{ message: string }> => {
  const res = await axiosLayer.get<ApiEnvelope<{ message: string }>>("/user/status");
  return unwrap<{ message: string }>(res);
};

export const logout = async (): Promise<void> => {
  await axiosLayer.post("/logout", null, {
    withCredentials: true,
  });
};

export interface SessionInfo {
  family_id: string;
  device_id: string;
  device_info: string;
  ip_address: string;
  created_at: string;
  expires_at: string;
}

export const getSessions = async (): Promise<SessionInfo[]> => {
  const res = await axiosLayer.get<ApiEnvelope<{ sessions: SessionInfo[] }>>("/user/me/sessions");
  return unwrap<{ sessions: SessionInfo[] }>(res).sessions;
};

export const revokeSession = async (family_id: string, password?: string): Promise<void> => {
  await axiosLayer.post<ApiEnvelope<never>>("/user/me/sessions/revoke", {
    family_id,
    password,
  });
};

export interface SetupStatusResponse {
  setup_required: boolean;
}

export const getSetupStatus = async (): Promise<SetupStatusResponse> => {
  const res = await axiosLayer.get<ApiEnvelope<SetupStatusResponse>>("/setup/status");
  return unwrap<SetupStatusResponse>(res);
};

export const setupAdmin = async (username: string, password: string): Promise<{ message: string }> => {
  const res = await axiosLayer.post<ApiEnvelope<{ message: string }>>(
    "/setup/admin",
    { username, password },
    { headers: { "Content-Type": "application/json" } }
  );
  return unwrap<{ message: string }>(res);
};