import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import axiosLayer from '@/api/axiosLayer';
import { getConfigs, updateConfig } from "@/api/api-config";
import { getMe } from "@/api/api-auth";
import { getAuditLogs } from "@/api/api-audit";
import type { AuditLogEntry } from "@/api/api-audit";
import type { ConfigItem } from "@/api/api-config";
import { ArrowLeft, Settings2, Users, ScrollText, Database } from "lucide-react";
import { useNavigate } from "react-router-dom";
import type { UserInfo } from './admin/types';
import ConfigsTab from './admin/ConfigsTab';
import UsersTab from './admin/UsersTab';
import AuditTab from './admin/AuditTab';

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
          <ConfigsTab
            configs={configs}
            loadingConfigs={loadingConfigs}
            savingConfigs={savingConfigs}
            hasConfigChanges={hasConfigChanges}
            onUpdateLocalConfig={updateLocalConfig}
            onSave={() => { void handleSaveConfigs(); }}
          />
        )}

        {activeTab === 'users' && (
          <UsersTab
            users={users}
            currentUsername={currentUsername}
            newUsername={newUsername}
            newPassword={newPassword}
            newRole={newRole}
            newMfaMandatory={newMfaMandatory}
            showPassword={showPassword}
            isCreatingUser={isCreatingUser}
            setNewUsername={setNewUsername}
            setNewPassword={setNewPassword}
            setNewRole={setNewRole}
            setNewMfaMandatory={setNewMfaMandatory}
            setShowPassword={setShowPassword}
            onCreateUser={handleCreateUser}
            onRevoke={handleRevoke}
            onDelete={handleDelete}
          />
        )}

        {activeTab === 'audit' && (
          <AuditTab
            auditLogs={auditLogs}
            auditTotal={auditTotal}
            auditPage={auditPage}
            auditPageSize={auditPageSize}
            auditEventType={auditEventType}
            auditUsername={auditUsername}
            auditUsernameInput={auditUsernameInput}
            loadingAudit={loadingAudit}
            setAuditEventType={setAuditEventType}
            setAuditPage={setAuditPage}
            setAuditUsername={setAuditUsername}
            setAuditUsernameInput={setAuditUsernameInput}
            onRefresh={() => { setAuditRefreshKey((k) => k + 1); }}
          />
        )}
      </div>
    </div>
  );
};

export default AdminPage;
