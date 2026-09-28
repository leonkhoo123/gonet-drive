export const AUDIT_EVENT_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'All Events' },
  { value: 'login_success', label: 'Login Success' },
  { value: 'login_failure', label: 'Login Failure' },
  { value: 'account_locked', label: 'Account Locked' },
  { value: 'mfa_setup_initiated', label: 'MFA Setup Initiated' },
  { value: 'mfa_enabled', label: 'MFA Enabled' },
  { value: 'mfa_verify_success', label: 'MFA Verify Success' },
  { value: 'mfa_verify_failure', label: 'MFA Verify Failure' },
  { value: 'mfa_lockout_triggered', label: 'MFA Lockout Triggered' },
  { value: 'mfa_recovery_success', label: 'MFA Recovery Success' },
  { value: 'mfa_recovery_failure', label: 'MFA Recovery Failure' },
  { value: 'token_compromise', label: 'Token Compromise' },
  { value: 'session_revoked', label: 'Session Revoked' },
  { value: 'all_sessions_revoked', label: 'All Sessions Revoked' },
  { value: 'logout', label: 'Logout' },
  { value: 'jwt_off_active', label: 'JWT-Off Active' },
  { value: 'admin_provisioned', label: 'Admin Provisioned' },
  { value: 'user_created', label: 'User Created' },
  { value: 'user_deleted', label: 'User Deleted' },
];

export const AUDIT_DANGER_EVENTS = new Set(['account_locked', 'mfa_lockout_triggered', 'token_compromise']);
export const AUDIT_WARNING_EVENTS = new Set(['login_failure', 'mfa_verify_failure', 'mfa_recovery_failure']);
export const AUDIT_SUCCESS_EVENTS = new Set([
  'login_success',
  'mfa_enabled',
  'mfa_verify_success',
  'mfa_recovery_success',
  'admin_provisioned',
  'user_created',
]);

export const auditEventLabel = (type: string): string =>
  AUDIT_EVENT_OPTIONS.find((o) => o.value === type)?.label ?? type;

export const auditEventBadgeClass = (type: string): string => {
  if (AUDIT_DANGER_EVENTS.has(type)) {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400';
  }
  if (AUDIT_WARNING_EVENTS.has(type)) {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400';
  }
  if (AUDIT_SUCCESS_EVENTS.has(type)) {
    return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400';
  }
  return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
};

export const formatAuditTime = (value: string): string => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
};
