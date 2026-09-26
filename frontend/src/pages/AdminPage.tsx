import { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Switch } from "@/components/ui/switch";
import { toast } from 'sonner';
import axiosLayer from '@/api/axiosLayer';
import { getConfigs, updateConfig } from "@/api/api-config";
import { getMe } from "@/api/api-auth";
import { getAuditLogs } from "@/api/api-audit";
import type { AuditLogEntry } from "@/api/api-audit";
import { getConfig } from '@/config';
import { Logo } from '@/components/Logo';
import type { ConfigItem } from "@/api/api-config";
import { 
  Loader2, 
  ArrowLeft, 
  Eye, 
  EyeOff, 
  Settings2, 
  Users, 
  ShieldCheck, 
  ShieldAlert,
  UserPlus,
  Trash2,
  KeyRound,
  Database,
  ScrollText,
  RefreshCw,
  ChevronLeft,
  ChevronRight
} from "lucide-react";
import { useNavigate } from "react-router-dom";

interface UserInfo {
  id: string;
  username: string;
  role: string;
  mfa_enabled: boolean;
  mfa_mandatory: boolean;
  locked_until?: string;
}

const AUDIT_EVENT_OPTIONS: { value: string; label: string }[] = [
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

const AUDIT_DANGER_EVENTS = new Set(['account_locked', 'mfa_lockout_triggered', 'token_compromise']);
const AUDIT_WARNING_EVENTS = new Set(['login_failure', 'mfa_verify_failure', 'mfa_recovery_failure']);
const AUDIT_SUCCESS_EVENTS = new Set([
  'login_success',
  'mfa_enabled',
  'mfa_verify_success',
  'mfa_recovery_success',
  'admin_provisioned',
  'user_created',
]);

const auditEventLabel = (type: string): string =>
  AUDIT_EVENT_OPTIONS.find((o) => o.value === type)?.label ?? type;

const auditEventBadgeClass = (type: string): string => {
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

const formatAuditTime = (value: string): string => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
};

const AdminPage = () => {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'configs' | 'users' | 'audit'>('configs');
  const [currentUsername, setCurrentUsername] = useState('');
  
  // Users state
  const [users, setUsers] = useState<UserInfo[]>([]);
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState('user');
  const [newMfaMandatory, setNewMfaMandatory] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [isCreatingUser, setIsCreatingUser] = useState(false);

  // Configs state
  const [configs, setConfigs] = useState<ConfigItem[]>([]);
  const [originalConfigs, setOriginalConfigs] = useState<ConfigItem[]>([]);
  const [loadingConfigs, setLoadingConfigs] = useState(false);
  const [savingConfigs, setSavingConfigs] = useState(false);
  
  // Logo state
  const [uploadingLogo, setUploadingLogo] = useState(false);
  const [logoUrl, setLogoUrl] = useState(() => `${getConfig().apiBaseUrl}/config/logo`);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Audit logs state
  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditPage, setAuditPage] = useState(1);
  const [auditPageSize] = useState(50);
  const [auditEventType, setAuditEventType] = useState('');
  const [auditUsername, setAuditUsername] = useState('');
  const [auditUsernameInput, setAuditUsernameInput] = useState('');
  const [loadingAudit, setLoadingAudit] = useState(false);
  const [auditRefreshKey, setAuditRefreshKey] = useState(0);

  useEffect(() => {
    getMe().then(res => {
      setCurrentUsername(res.username);
    }).catch(console.error);
  }, []);

  useEffect(() => {
    if (activeTab === 'users') {
      void fetchUsers();
      return;
    }
    if (activeTab === 'audit') {
      let cancelled = false;
      setLoadingAudit(true);
      getAuditLogs({
        page: auditPage,
        page_size: auditPageSize,
        event_type: auditEventType || undefined,
        username: auditUsername || undefined,
      })
        .then((data) => {
          if (cancelled) return;
          setAuditLogs(data.logs);
          setAuditTotal(data.total);
        })
        .catch(() => {
          if (!cancelled) toast.error('Failed to fetch audit logs');
        })
        .finally(() => {
          if (!cancelled) setLoadingAudit(false);
        });
      return () => {
        cancelled = true;
      };
    }
    void fetchConfigs();
  }, [activeTab, auditPage, auditPageSize, auditEventType, auditUsername, auditRefreshKey]);

  const fetchUsers = async () => {
    try {
      const res = await axiosLayer.get<{ status: string; data: { users: UserInfo[] } }>('/user/admin/users');
      setUsers(res.data.data.users);
    } catch {
      toast.error('Failed to fetch users');
    }
  };

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsCreatingUser(true);
      await axiosLayer.post('/user/admin/users', {
        username: newUsername,
        password: newPassword,
        role: newRole,
        mfa_mandatory: newMfaMandatory
      });
      toast.success('User created successfully');
      setNewUsername('');
      setNewPassword('');
      setNewMfaMandatory(false);
      setNewRole('user');
      void fetchUsers();
    } catch (err) {
      const errorMsg = (err as { response?: { data?: { error?: string } } }).response?.data?.error ?? 'Failed to create user';
      toast.error(errorMsg);
    } finally {
      setIsCreatingUser(false);
    }
  };

  const handleRevoke = async (id: string) => {
    try {
      await axiosLayer.post(`/user/admin/users/${id}/revoke-all`);
      toast.success('Sessions revoked successfully');
    } catch {
      toast.error('Failed to revoke sessions');
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this user? This action cannot be undone.')) return;
    try {
      await axiosLayer.delete(`/user/admin/users/${id}`);
      toast.success('User deleted successfully');
      void fetchUsers();
    } catch {
      toast.error('Failed to delete user');
    }
  };

  const fetchConfigs = async () => {
    try {
      setLoadingConfigs(true);
      const data = await getConfigs();
      setConfigs(data);
      setOriginalConfigs(JSON.parse(JSON.stringify(data)) as ConfigItem[]);
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : "Failed to fetch configurations");
    } finally {
      setLoadingConfigs(false);
    }
  };

  const handleSaveConfigs = async () => {
    try {
      setSavingConfigs(true);
      const updates = [];

      for (const current of configs) {
        const original = originalConfigs.find((c) => c.id === current.id);
        if (!original) continue;

        const hasChanged = 
          current.config_value !== original.config_value || 
          current.is_enabled !== original.is_enabled;

        if (hasChanged) {
          updates.push(
            updateConfig(current.id, {
              config_value: current.config_value,
              is_enabled: current.is_enabled,
            })
          );
        }
      }

      if (updates.length > 0) {
        await Promise.all(updates);
        toast.success(`Successfully updated ${String(updates.length)} configuration(s)`);
        await fetchConfigs();
      } else {
        toast.info("No changes to save");
      }
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : "Failed to update configurations");
    } finally {
      setSavingConfigs(false);
    }
  };

  const updateLocalConfig = (id: number, field: keyof ConfigItem, value: string | boolean | null) => {
    setConfigs((prev) =>
      prev.map((c) => (c.id === id ? { ...c, [field]: value } : c))
    );
  };

  const handleLogoUpload = async () => {
    if (!selectedFile) return;

    try {
      setUploadingLogo(true);
      const formData = new FormData();
      formData.append('logo', selectedFile);
      
      await axiosLayer.put('/user/admin/config/logo', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      
      toast.success('Logo updated successfully');
      setLogoUrl(`${getConfig().apiBaseUrl}/config/logo?t=${String(Date.now())}`);
      setSelectedFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } catch (err) {
      const errorMsg = (err as { response?: { data?: { error?: string } } }).response?.data?.error ?? 'Failed to update logo';
      toast.error(errorMsg);
    } finally {
      setUploadingLogo(false);
    }
  };

  const handleCancelLogo = () => {
    setSelectedFile(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const hasConfigChanges = JSON.stringify(configs) !== JSON.stringify(originalConfigs);

  return (
    <div className="h-full bg-background text-foreground flex flex-col">
      <div className="h-16 md:h-14 border-b flex items-center px-4 md:px-6 shrink-0 gap-4 bg-card">
        <Button variant="ghost" size="icon" onClick={() => { void navigate(-1); }} className="rounded-full">
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Database className="h-5 w-5 text-primary" />
            Manage Cloud
          </h1>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 md:p-8 space-y-6 max-w-6xl mx-auto w-full">
        <div className="flex space-x-2 border-b">
          <button
            className={`px-4 py-3 font-medium text-sm transition-colors border-b-2 flex items-center gap-2 ${
              activeTab === 'configs'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => { setActiveTab('configs'); }}
          >
            <Settings2 className="h-4 w-4" />
            Configurations
          </button>
          <button
            className={`px-4 py-3 font-medium text-sm transition-colors border-b-2 flex items-center gap-2 ${
              activeTab === 'users'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => { setActiveTab('users'); }}
          >
            <Users className="h-4 w-4" />
            Users
          </button>
          <button
            className={`px-4 py-3 font-medium text-sm transition-colors border-b-2 flex items-center gap-2 ${
              activeTab === 'audit'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => { setActiveTab('audit'); }}
          >
            <ScrollText className="h-4 w-4" />
            Auth Logs
          </button>
        </div>

        {activeTab === 'configs' && (
          <div className="space-y-4">
            <Card className="flex flex-col border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
              <CardHeader className="bg-muted/30 pb-3 px-4 md:px-6">
                <CardTitle>Site Logo</CardTitle>
                <CardDescription className="hidden sm:block">Upload a custom logo for your cloud instance. Recommended 1:1 aspect ratio, PNG only.</CardDescription>
              </CardHeader>
              <CardContent className="pt-0 md:pt-6 px-4 md:px-6 flex flex-row flex-wrap items-center gap-4 sm:gap-6">
                <div className="bg-muted rounded-lg p-4 flex items-center justify-center border w-32 h-32 shrink-0">
                  <Logo src={logoUrl} className="max-w-full max-h-full object-contain" />
                </div>
                <div className="space-y-4 flex-1">
                  <div className="flex flex-col sm:flex-row sm:items-center gap-4">
                    <Label className="text-sm font-medium">Choose file to upload :</Label>
                    <input
                      type="file"
                      accept="image/png"
                      className="hidden"
                      ref={fileInputRef}
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          if (file.type !== 'image/png') {
                            toast.error('Only PNG files are allowed');
                            e.target.value = '';
                            return;
                          }
                          if (file.size > 5 * 1024 * 1024) {
                            toast.error('File size exceeds 5MB limit');
                            e.target.value = '';
                            return;
                          }
                          setSelectedFile(file);
                        }
                      }}
                    />
                    
                    {!selectedFile ? (
                      <Button 
                        variant="outline" 
                        onClick={() => fileInputRef.current?.click()}
                        disabled={uploadingLogo}
                      >
                        Browse
                      </Button>
                    ) : (
                      <div className="flex items-center gap-2 flex-1 w-full max-w-md">
                        <Input 
                          value={selectedFile.name} 
                          readOnly 
                          className="flex-1 text-muted-foreground bg-muted/50 truncate" 
                        />
                        <Button 
                          onClick={() => { void handleLogoUpload(); }} 
                          disabled={uploadingLogo}
                        >
                          {uploadingLogo && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                          Upload
                        </Button>
                        <Button 
                          variant="outline" 
                          onClick={handleCancelLogo}
                          disabled={uploadingLogo}
                        >
                          Cancel
                        </Button>
                      </div>
                    )}
                  </div>
                  {uploadingLogo && <p className="text-sm text-muted-foreground flex items-center"><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Uploading...</p>}
                </div>
              </CardContent>
            </Card>

            <Card className="flex flex-col border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
              <CardHeader className="flex flex-row items-center justify-between bg-muted/30 pb-3 px-4 md:px-6">
                <div>
                  <CardTitle>System Configurations</CardTitle>
                  <CardDescription className="hidden sm:block">Manage global settings and features for your cloud instance.</CardDescription>
                </div>
                <Button 
                  onClick={() => { void handleSaveConfigs(); }} 
                  disabled={!hasConfigChanges || savingConfigs || loadingConfigs}
                  className="bg-primary hover:bg-primary/90 text-primary-foreground shadow-sm"
                >
                  {savingConfigs && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Save Changes
                </Button>
              </CardHeader>
              <CardContent className="pt-0 md:pt-6 px-4 md:px-6">
                {loadingConfigs && configs.length === 0 ? (
                  <div className="flex items-center justify-center p-12">
                    <Loader2 className="h-8 w-8 animate-spin text-primary" />
                  </div>
                ) : configs.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground rounded-md border">
                    <Settings2 className="h-8 w-8 mx-auto mb-3 opacity-20" />
                    No configurations found
                  </div>
                ) : (
                  <>
                    <div className="hidden md:block rounded-md border overflow-hidden">
                      <Table>
                        <TableHeader className="bg-muted/50">
                          <TableRow>
                            <TableHead className="font-semibold">Configuration</TableHead>
                            <TableHead className="font-semibold">Value</TableHead>
                            <TableHead className="w-[120px] font-semibold text-center">Enabled</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {configs.map((config) => (
                            <TableRow key={config.id} className="hover:bg-muted/30 transition-colors">
                              <TableCell className="py-3">
                                <div className="font-medium text-base">{config.config_name}</div>
                                <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                                  <span className="bg-muted px-2 py-0.5 rounded-full font-mono">{config.config_type}</span>
                                  {config.config_unit && (
                                    <span className="text-muted-foreground/80">Unit: {config.config_unit}</span>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell className="py-3">
                                <Input
                                  value={config.config_value ?? ""}
                                  onChange={(e) => {
                                    updateLocalConfig(config.id, "config_value", e.target.value);
                                  }}
                                  placeholder="Value"
                                  disabled={savingConfigs}
                                  className="focus-visible:ring-ring max-w-md"
                                />
                              </TableCell>
                              <TableCell className="py-3">
                                <div className="flex justify-center">
                                  <Switch
                                    checked={config.is_enabled}
                                    onCheckedChange={(checked) => {
                                      updateLocalConfig(config.id, "is_enabled", checked);
                                    }}
                                    disabled={savingConfigs}
                                    className="data-[state=checked]:bg-primary"
                                  />
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>

                    <div className="md:hidden space-y-2">
                      {configs.map((config) => (
                        <div key={config.id} className="rounded-md border p-3 space-y-2 bg-card">
                          <div>
                            <div className="font-medium text-base">{config.config_name}</div>
                            <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                              <span className="bg-muted px-2 py-0.5 rounded-full font-mono">{config.config_type}</span>
                              {config.config_unit && (
                                <span className="text-muted-foreground/80">Unit: {config.config_unit}</span>
                              )}
                            </div>
                          </div>
                          <Input
                            value={config.config_value ?? ""}
                            onChange={(e) => {
                              updateLocalConfig(config.id, "config_value", e.target.value);
                            }}
                            placeholder="Value"
                            disabled={savingConfigs}
                            className="focus-visible:ring-ring"
                          />
                          <div className="flex items-center justify-between">
                            <span className="text-sm text-muted-foreground">Enabled</span>
                            <Switch
                              checked={config.is_enabled}
                              onCheckedChange={(checked) => {
                                updateLocalConfig(config.id, "is_enabled", checked);
                              }}
                              disabled={savingConfigs}
                              className="data-[state=checked]:bg-primary"
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        )}

        {activeTab === 'users' && (
          <div className="space-y-4">
            <Card className="border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
              <CardHeader className="pb-3 px-4 md:px-6">
                <CardTitle className="flex items-center gap-2">
                  <UserPlus className="h-5 w-5 text-primary" />
                  Create User
                </CardTitle>
                <CardDescription>Create accounts for friends and family without sharing your admin password. Revoke access anytime.</CardDescription>
              </CardHeader>
              <CardContent className="pt-0 px-4 md:px-6">
                <form onSubmit={(e) => { void handleCreateUser(e); }} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 items-end">
                  <div className="space-y-2">
                    <Label htmlFor="username">Username</Label>
                    <Input 
                      id="username"
                      required 
                      value={newUsername} 
                      onChange={e => { setNewUsername(e.target.value); }} 
                      placeholder="e.g. jdoe"
                      className="focus-visible:ring-ring"
                    />
                  </div>
                  
                  <div className="space-y-2">
                    <Label htmlFor="password">Password</Label>
                    <div className="relative">
                      <Input 
                        id="password"
                        type={showPassword ? "text" : "password"} 
                        required 
                        value={newPassword} 
                        onChange={e => { setNewPassword(e.target.value); }} 
                        placeholder="••••••••"
                        className="pr-10 focus-visible:ring-ring"
                      />
                      <button
                        type="button"
                        onClick={() => { setShowPassword(!showPassword); }}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors focus:outline-none"
                        tabIndex={-1}
                      >
                        {showPassword ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                  </div>
                  
                  <div className="space-y-2">
                    <Label htmlFor="role">Role</Label>
                    <select 
                      id="role"
                      className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50" 
                      value={newRole} 
                      onChange={e => { setNewRole(e.target.value); }}
                    >
                      <option value="user">User</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                  
                  <div className="space-y-3 flex flex-row items-center justify-between rounded-lg border p-3 shadow-sm md:col-span-2 lg:col-span-2 bg-muted/20">
                    <div className="space-y-0.5">
                      <Label className="text-base font-medium">Require MFA</Label>
                      <p className="text-xs text-muted-foreground">
                        Force this user to setup Multi-Factor Authentication
                      </p>
                    </div>
                    <Switch 
                      checked={newMfaMandatory} 
                      onCheckedChange={setNewMfaMandatory}
                      className="data-[state=checked]:bg-primary"
                    />
                  </div>
                  
                  <div className="flex justify-end md:col-span-2 lg:col-span-1">
                    <Button 
                      type="submit" 
                      className="w-full md:w-auto bg-primary hover:bg-primary/90 text-primary-foreground"
                      disabled={isCreatingUser || !newUsername || !newPassword}
                    >
                      {isCreatingUser ? (
                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      ) : (
                        <UserPlus className="h-4 w-4 mr-2" />
                      )}
                      Create User
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>

            <Card className="border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
              <CardHeader className="bg-muted/30 pb-3 px-4 md:px-6">
                <CardTitle className="flex items-center gap-2">
                  <Users className="h-5 w-5 text-primary" />
                  User List
                </CardTitle>
                <CardDescription className="hidden sm:block">Manage existing users and their access.</CardDescription>
              </CardHeader>
              <CardContent className="pt-0 md:pt-6 px-4 md:px-6">
                {users.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground rounded-md border">
                    <Users className="h-8 w-8 mx-auto mb-3 opacity-20" />
                    No users found
                  </div>
                ) : (
                  <>
                    <div className="hidden md:block rounded-md border overflow-x-auto">
                      <Table>
                        <TableHeader className="bg-muted/50">
                          <TableRow>
                            <TableHead className="font-semibold">Username</TableHead>
                            <TableHead className="font-semibold">Role</TableHead>
                            <TableHead className="font-semibold">Security</TableHead>
                            <TableHead className="font-semibold text-center">Status</TableHead>
                            <TableHead className="font-semibold text-right">Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {users.map(u => (
                            <TableRow key={u.id} className="hover:bg-muted/30 transition-colors">
                              <TableCell className="font-medium">
                                <div className="flex items-center gap-2">
                                  <div className="h-8 w-8 rounded-full bg-primary/10 text-primary flex items-center justify-center font-semibold text-xs">
                                    {u.username.substring(0, 2).toUpperCase()}
                                  </div>
                                  {u.username}
                                  {u.username === currentUsername && (
                                    <span className="text-[10px] bg-primary/10 text-primary px-2 py-0.5 rounded-full ml-2">
                                      You
                                    </span>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell>
                                <span className={`text-xs px-2.5 py-1 rounded-full font-medium ${
                                  u.role === 'admin' 
                                    ? 'bg-primary/10 text-primary'
                                    : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                                }`}>
                                  {u.role.charAt(0).toUpperCase() + u.role.slice(1)}
                                </span>
                              </TableCell>
                              <TableCell>
                                <div className="flex flex-col gap-1 text-xs">
                                  <div className="flex items-center gap-1.5">
                                    {u.mfa_enabled ? (
                                      <ShieldCheck className="h-3.5 w-3.5 text-green-500" />
                                    ) : (
                                      <ShieldAlert className="h-3.5 w-3.5 text-amber-500" />
                                    )}
                                    <span className={u.mfa_enabled ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'}>
                                      MFA {u.mfa_enabled ? 'Enabled' : 'Disabled'}
                                    </span>
                                  </div>
                                  <div className="text-muted-foreground flex items-center gap-1.5 pl-5">
                                    {u.mfa_mandatory ? 'Required' : 'Optional'}
                                  </div>
                                </div>
                              </TableCell>
                              <TableCell className="text-center">
                                {u.locked_until ? (
                                  <span className="inline-flex items-center gap-1 text-xs font-medium text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400 px-2.5 py-1 rounded-full">
                                    <ShieldAlert className="h-3 w-3" />
                                    Locked
                                  </span>
                                ) : (
                                  <span className="inline-flex items-center gap-1 text-xs font-medium text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400 px-2.5 py-1 rounded-full">
                                    <ShieldCheck className="h-3 w-3" />
                                    Active
                                  </span>
                                )}
                              </TableCell>
                              <TableCell className="text-right">
                                <div className="flex justify-end gap-2">
                                  <Button 
                                    variant="outline" 
                                    size="sm" 
                                    onClick={() => { void handleRevoke(u.id); }}
                                    disabled={u.username === currentUsername || u.role === 'admin'}
                                    className="h-8 text-xs px-2.5"
                                    title="Revoke all active sessions for this user"
                                  >
                                    <KeyRound className="h-3.5 w-3.5 mr-1" />
                                    Revoke
                                  </Button>
                                  <Button 
                                    variant="destructive" 
                                    size="sm" 
                                    onClick={() => { void handleDelete(u.id); }}
                                    disabled={u.username === currentUsername}
                                    className="h-8 text-xs px-2.5"
                                  >
                                    <Trash2 className="h-3.5 w-3.5" />
                                  </Button>
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>

                    <div className="md:hidden space-y-2">
                      {users.map(u => (
                        <div key={u.id} className="rounded-md border p-3 space-y-2 bg-card">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2 min-w-0">
                              <div className="h-8 w-8 rounded-full bg-primary/10 text-primary flex items-center justify-center font-semibold text-xs shrink-0">
                                {u.username.substring(0, 2).toUpperCase()}
                              </div>
                              <div className="min-w-0">
                                <span className="font-medium truncate block">{u.username}</span>
                                {u.username === currentUsername && (
                                  <span className="text-[10px] bg-primary/10 text-primary px-1.5 py-0.5 rounded-full inline-block mt-0.5">
                                    You
                                  </span>
                                )}
                              </div>
                            </div>
                            <span className={`text-xs px-2.5 py-1 rounded-full font-medium shrink-0 ${
                              u.role === 'admin' 
                                ? 'bg-primary/10 text-primary'
                                : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                            }`}>
                              {u.role.charAt(0).toUpperCase() + u.role.slice(1)}
                            </span>
                          </div>
                          
                          <div className="grid grid-cols-2 gap-1.5 text-sm">
                            <div>
                              <span className="text-xs text-muted-foreground block">MFA</span>
                              <div className="flex items-center gap-1 mt-0.5">
                                {u.mfa_enabled ? (
                                  <ShieldCheck className="h-3.5 w-3.5 text-green-500" />
                                ) : (
                                  <ShieldAlert className="h-3.5 w-3.5 text-amber-500" />
                                )}
                                <span className={`text-xs ${u.mfa_enabled ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'}`}>
                                  {u.mfa_enabled ? 'On' : 'Off'}
                                </span>
                              </div>
                              <span className="text-[10px] text-muted-foreground">{u.mfa_mandatory ? 'Required' : 'Optional'}</span>
                            </div>
                            <div>
                              <span className="text-xs text-muted-foreground block">Status</span>
                              {u.locked_until ? (
                                <span className="inline-flex items-center gap-1 text-xs font-medium text-red-600 mt-0.5">
                                  <ShieldAlert className="h-3 w-3" />
                                  Locked
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1 text-xs font-medium text-green-600 mt-0.5">
                                  <ShieldCheck className="h-3 w-3" />
                                  Active
                                </span>
                              )}
                            </div>
                          </div>
                          
                          <div className="flex gap-2 pt-1.5 border-t border-border">
                            <Button 
                              variant="outline" 
                              size="sm" 
                              onClick={() => { void handleRevoke(u.id); }}
                              disabled={u.username === currentUsername || u.role === 'admin'}
                              className="h-8 text-xs px-2.5 flex-1"
                            >
                              <KeyRound className="h-3.5 w-3.5 mr-1" />
                              Revoke
                            </Button>
                            <Button 
                              variant="destructive" 
                              size="sm" 
                              onClick={() => { void handleDelete(u.id); }}
                              disabled={u.username === currentUsername}
                              className="h-8 text-xs px-2.5 flex-1"
                            >
                              <Trash2 className="h-3.5 w-3.5 mr-1" />
                              Delete
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        )}

        {activeTab === 'audit' && (
          <div className="space-y-4">
            <Card className="border-border shadow-sm py-3 md:py-6 gap-3 md:gap-6">
              <CardHeader className="bg-muted/30 pb-3 px-4 md:px-6">
                <CardTitle className="flex items-center gap-2">
                  <ScrollText className="h-5 w-5 text-primary" />
                  Auth Logs
                </CardTitle>
                <CardDescription className="hidden sm:block">
                  Security events captured by the authentication service: logins, logouts, MFA, session and account changes.
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-0 md:pt-6 px-4 md:px-6 space-y-4">
                <div className="flex flex-col sm:flex-row gap-2">
                  <select
                    className="flex h-10 w-full sm:w-60 items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    value={auditEventType}
                    onChange={(e) => {
                      setAuditEventType(e.target.value);
                      setAuditPage(1);
                    }}
                  >
                    {AUDIT_EVENT_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>{opt.label}</option>
                    ))}
                  </select>
                  <form
                    className="flex flex-1 gap-2"
                    onSubmit={(e) => {
                      e.preventDefault();
                      setAuditPage(1);
                      setAuditUsername(auditUsernameInput.trim());
                    }}
                  >
                    <Input
                      value={auditUsernameInput}
                      onChange={(e) => { setAuditUsernameInput(e.target.value); }}
                      placeholder="Filter by username"
                      className="flex-1 focus-visible:ring-ring"
                    />
                    <Button type="submit" variant="outline">Search</Button>
                    {auditUsername && (
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() => {
                          setAuditUsernameInput('');
                          setAuditUsername('');
                          setAuditPage(1);
                        }}
                      >
                        Clear
                      </Button>
                    )}
                  </form>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => { setAuditRefreshKey((k) => k + 1); }}
                    disabled={loadingAudit}
                  >
                    <RefreshCw className={`h-4 w-4 mr-2 ${loadingAudit ? 'animate-spin' : ''}`} />
                    Refresh
                  </Button>
                </div>

                {loadingAudit && auditLogs.length === 0 ? (
                  <div className="flex items-center justify-center p-12">
                    <Loader2 className="h-8 w-8 animate-spin text-primary" />
                  </div>
                ) : auditLogs.length === 0 ? (
                  <div className="text-center py-12 text-muted-foreground rounded-md border">
                    <ScrollText className="h-8 w-8 mx-auto mb-3 opacity-20" />
                    No audit events found
                  </div>
                ) : (
                  <>
                    <div className="hidden md:block rounded-md border overflow-x-auto">
                      <Table>
                        <TableHeader className="bg-muted/50">
                          <TableRow>
                            <TableHead className="font-semibold">Time</TableHead>
                            <TableHead className="font-semibold">Event</TableHead>
                            <TableHead className="font-semibold">User</TableHead>
                            <TableHead className="font-semibold">IP Address</TableHead>
                            <TableHead className="font-semibold">Device</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {auditLogs.map((log) => (
                            <TableRow key={log.id} className="hover:bg-muted/30 transition-colors">
                              <TableCell className="py-3 text-xs text-muted-foreground whitespace-nowrap">
                                {formatAuditTime(log.timestamp)}
                              </TableCell>
                              <TableCell className="py-3">
                                <span className={`text-xs px-2.5 py-1 rounded-full font-medium whitespace-nowrap ${auditEventBadgeClass(log.event_type)}`}>
                                  {auditEventLabel(log.event_type)}
                                </span>
                              </TableCell>
                              <TableCell className="py-3 font-medium">{log.username || '—'}</TableCell>
                              <TableCell className="py-3 font-mono text-xs">{log.ip_address || '—'}</TableCell>
                              <TableCell className="py-3 text-xs text-muted-foreground max-w-[280px] truncate" title={log.device_info}>
                                {log.device_info || '—'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>

                    <div className="md:hidden space-y-2">
                      {auditLogs.map((log) => (
                        <div key={log.id} className="rounded-md border p-3 space-y-2 bg-card">
                          <div className="flex items-center justify-between gap-2">
                            <span className={`text-xs px-2.5 py-1 rounded-full font-medium ${auditEventBadgeClass(log.event_type)}`}>
                              {auditEventLabel(log.event_type)}
                            </span>
                            <span className="text-[11px] text-muted-foreground shrink-0">{formatAuditTime(log.timestamp)}</span>
                          </div>
                          <div className="grid grid-cols-2 gap-1.5 text-sm">
                            <div>
                              <span className="text-xs text-muted-foreground block">User</span>
                              <span className="truncate block">{log.username || '—'}</span>
                            </div>
                            <div>
                              <span className="text-xs text-muted-foreground block">IP</span>
                              <span className="font-mono text-xs truncate block">{log.ip_address || '—'}</span>
                            </div>
                          </div>
                          {log.device_info && (
                            <div className="text-[11px] text-muted-foreground break-words border-t border-border pt-1.5">
                              {log.device_info}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>

                    <div className="flex items-center justify-between pt-2">
                      <span className="text-sm text-muted-foreground">
                        Page {auditPage} of {Math.max(1, Math.ceil(auditTotal / auditPageSize))} · {auditTotal} event{auditTotal === 1 ? '' : 's'}
                      </span>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => { setAuditPage((p) => Math.max(1, p - 1)); }}
                          disabled={auditPage <= 1 || loadingAudit}
                        >
                          <ChevronLeft className="h-4 w-4 mr-1" />
                          Prev
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => { setAuditPage((p) => p + 1); }}
                          disabled={auditPage >= Math.ceil(auditTotal / auditPageSize) || loadingAudit}
                        >
                          Next
                          <ChevronRight className="h-4 w-4 ml-1" />
                        </Button>
                      </div>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
};

export default AdminPage;
