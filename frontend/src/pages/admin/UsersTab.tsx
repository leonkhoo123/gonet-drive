import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Switch } from "@/components/ui/switch";
import type { UserInfo } from './types';
import {
  Eye,
  EyeOff,
  Users,
  ShieldCheck,
  ShieldAlert,
  UserPlus,
  Trash2,
  KeyRound,
  Loader2,
} from "lucide-react";

interface UsersTabProps {
  users: UserInfo[];
  currentUsername: string;
  newUsername: string;
  newPassword: string;
  newRole: string;
  newMfaMandatory: boolean;
  showPassword: boolean;
  isCreatingUser: boolean;
  setNewUsername: (value: string) => void;
  setNewPassword: (value: string) => void;
  setNewRole: (value: string) => void;
  setNewMfaMandatory: (value: boolean) => void;
  setShowPassword: (value: boolean) => void;
  onCreateUser: (e: React.FormEvent) => Promise<void>;
  onRevoke: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

const UsersTab = ({
  users,
  currentUsername,
  newUsername,
  newPassword,
  newRole,
  newMfaMandatory,
  showPassword,
  isCreatingUser,
  setNewUsername,
  setNewPassword,
  setNewRole,
  setNewMfaMandatory,
  setShowPassword,
  onCreateUser,
  onRevoke,
  onDelete,
}: UsersTabProps) => {
  return (
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
          <form onSubmit={(e) => { void onCreateUser(e); }} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 items-end">
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
                              onClick={() => { void onRevoke(u.id); }}
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
                              onClick={() => { void onDelete(u.id); }}
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
                        onClick={() => { void onRevoke(u.id); }}
                        disabled={u.username === currentUsername || u.role === 'admin'}
                        className="h-8 text-xs px-2.5 flex-1"
                      >
                        <KeyRound className="h-3.5 w-3.5 mr-1" />
                        Revoke
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => { void onDelete(u.id); }}
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
  );
};

export default UsersTab;
