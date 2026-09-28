import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import type { AuditLogEntry } from "@/api/api-audit";
import { AUDIT_EVENT_OPTIONS, auditEventLabel, auditEventBadgeClass, formatAuditTime } from './auditUtils';
import { Loader2, ScrollText, RefreshCw, ChevronLeft, ChevronRight } from "lucide-react";

interface AuditTabProps {
  auditLogs: AuditLogEntry[];
  auditTotal: number;
  auditPage: number;
  auditPageSize: number;
  auditEventType: string;
  auditUsername: string;
  auditUsernameInput: string;
  loadingAudit: boolean;
  setAuditEventType: (value: string) => void;
  setAuditPage: React.Dispatch<React.SetStateAction<number>>;
  setAuditUsername: (value: string) => void;
  setAuditUsernameInput: (value: string) => void;
  onRefresh: () => void;
}

const AuditTab = ({
  auditLogs,
  auditTotal,
  auditPage,
  auditPageSize,
  auditEventType,
  auditUsername,
  auditUsernameInput,
  loadingAudit,
  setAuditEventType,
  setAuditPage,
  setAuditUsername,
  setAuditUsernameInput,
  onRefresh,
}: AuditTabProps) => {
  return (
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
              onClick={onRefresh}
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
  );
};

export default AuditTab;
