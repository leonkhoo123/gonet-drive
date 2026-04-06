import { useState, useEffect } from 'react';
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
  Database
} from "lucide-react";
import { useNavigate } from "react-router-dom";

interface UserInfo {
  id: string;
  username: string;
  role: string;
  mfa_enabled: boolean;
  mfa_mandatory: boolean;
  storage_quota: number;
  failed_attempts: number;
  locked_until?: string;
  is_super_admin: boolean;
}

const AdminPage = () => {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'configs' | 'users'>('configs');
  const [isSuperAdmin, setIsSuperAdmin] = useState(false);
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

  useEffect(() => {
    getMe().then(res => {
      setIsSuperAdmin(res.is_super_admin);
      setCurrentUsername(res.username);
    }).catch(console.error);

    if (activeTab === 'users') {
      void fetchUsers();
    } else {
      void fetchConfigs();
    }
  }, [activeTab]);

  const fetchUsers = async () => {
    try {
      const res = await axiosLayer.get<UserInfo[]>('/user/admin/users');
      setUsers(res.data);
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
        mfa_mandatory: newMfaMandatory,
        storage_quota: 0
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
      await axiosLayer.post(`/user/admin/users/${id}/revoke`);
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
    } catch (error: any) {
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
        toast.success(`Successfully updated ${updates.length} configuration(s)`);
        await fetchConfigs();
      } else {
        toast.info("No changes to save");
      }
    } catch (error: any) {
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

  const hasConfigChanges = JSON.stringify(configs) !== JSON.stringify(originalConfigs);

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <div className="h-16 md:h-14 border-b flex items-center px-4 md:px-6 shrink-0 gap-4 bg-card">
        <Button variant="ghost" size="icon" onClick={() => navigate(-1)} className="rounded-full">
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <Database className="h-5 w-5 text-primary" />
            Manage Cloud
          </h1>
        </div>
      </div>

      <div className="flex-1 p-4 md:p-8 space-y-6 max-w-6xl mx-auto w-full">
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
        </div>

        {activeTab === 'configs' && (
          <Card className="flex flex-col border-border shadow-sm">
            <CardHeader className="flex flex-row items-center justify-between bg-muted/30 pb-4">
              <div>
                <CardTitle>System Configurations</CardTitle>
                <CardDescription>Manage global settings and features for your cloud instance.</CardDescription>
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
            <CardContent className="pt-6">
              {loadingConfigs && configs.length === 0 ? (
                <div className="flex items-center justify-center p-12">
                  <Loader2 className="h-8 w-8 animate-spin text-primary" />
                </div>
              ) : (
                <div className="rounded-md border overflow-hidden">
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
                          <TableCell className="py-4">
                            <div className="font-medium text-base">{config.config_name}</div>
                            <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                              <span className="bg-muted px-2 py-0.5 rounded-full font-mono">{config.config_type}</span>
                              {config.config_unit && (
                                <span className="text-muted-foreground/80">Unit: {config.config_unit}</span>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="py-4">
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
                          <TableCell className="py-4">
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
                      {configs.length === 0 && !loadingConfigs && (
                        <TableRow>
                          <TableCell colSpan={3} className="text-center py-12 text-muted-foreground">
                            <Settings2 className="h-8 w-8 mx-auto mb-3 opacity-20" />
                            No configurations found
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {activeTab === 'users' && (
          <div className="space-y-6">
            <Card className="border-border shadow-sm">
              <CardHeader className="bg-muted/30 pb-4">
                <CardTitle className="flex items-center gap-2">
                  <UserPlus className="h-5 w-5 text-primary" />
                  Create User
                </CardTitle>
                <CardDescription>Add a new user to your cloud instance.</CardDescription>
              </CardHeader>
              <CardContent className="pt-6">
                <form onSubmit={handleCreateUser} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 items-end">
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
                  
                  {isSuperAdmin && (
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
                  )}
                  
                  <div className={`flex justify-end ${isSuperAdmin ? 'md:col-span-2 lg:col-span-1' : 'md:col-span-2 lg:col-span-3'}`}>
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

            <Card className="border-border shadow-sm">
              <CardHeader className="bg-muted/30 pb-4">
                <CardTitle className="flex items-center gap-2">
                  <Users className="h-5 w-5 text-primary" />
                  User List
                </CardTitle>
                <CardDescription>Manage existing users and their access.</CardDescription>
              </CardHeader>
              <CardContent className="pt-6">
                <div className="rounded-md border overflow-x-auto">
                  <Table>
                    <TableHeader className="bg-muted/50">
                      <TableRow>
                        <TableHead className="font-semibold">Username</TableHead>
                        <TableHead className="font-semibold">Role</TableHead>
                        <TableHead className="font-semibold">Security</TableHead>
                        <TableHead className="font-semibold text-right">Quota</TableHead>
                        <TableHead className="font-semibold text-center">Status</TableHead>
                        <TableHead className="font-semibold text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {users.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={6} className="text-center py-12 text-muted-foreground">
                            <Users className="h-8 w-8 mx-auto mb-3 opacity-20" />
                            No users found
                          </TableCell>
                        </TableRow>
                      ) : (
                        users.map(u => (
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
                                u.is_super_admin || u.role === 'superadmin' 
                                  ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400' 
                                  : u.role === 'admin' 
                                    ? 'bg-primary/10 text-primary'
                                    : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                              }`}>
                                {u.is_super_admin ? 'Super Admin' : u.role.charAt(0).toUpperCase() + u.role.slice(1)}
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
                            <TableCell className="text-right font-mono text-sm">
                              {u.storage_quota.toLocaleString()}
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
                                  disabled={u.username === currentUsername || u.is_super_admin || (u.role === 'admin' && !isSuperAdmin) || (u.role === 'superadmin' && !isSuperAdmin)}
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
                                  disabled={u.username === currentUsername || u.is_super_admin || (u.role === 'admin' && !isSuperAdmin) || (u.role === 'superadmin' && !isSuperAdmin)}
                                  className="h-8 text-xs px-2.5"
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
};

export default AdminPage;
