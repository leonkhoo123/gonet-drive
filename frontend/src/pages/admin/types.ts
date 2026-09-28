export interface UserInfo {
  id: string;
  username: string;
  role: string;
  mfa_enabled: boolean;
  mfa_mandatory: boolean;
  locked_until?: string;
}
