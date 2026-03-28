import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getSessions, revokeSession, type SessionInfo } from "@/api/api-auth";
import { toast } from "sonner";
import { Loader2, MonitorSmartphone, Trash2, Eye, EyeOff } from "lucide-react";
import fpPromise from '@fingerprintjs/fingerprintjs';

interface DevicesModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function getOSFromUserAgent(userAgent: string): string {
  if (!userAgent) return "Unknown OS";
  
  const ua = userAgent.toLowerCase();
  
  if (ua.includes("iphone") || ua.includes("ipad") || ua.includes("ipod")) {
    return "iOS";
  } else if (ua.includes("mac os") || ua.includes("macintosh")) {
    return "macOS";
  } else if (ua.includes("android")) {
    return "Android";
  } else if (ua.includes("windows")) {
    return "Windows";
  } else if (ua.includes("linux")) {
    return "Linux";
  }
  
  return "Unknown OS";
}

export function DevicesModal({ open, onOpenChange }: DevicesModalProps) {
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [currentDeviceId, setCurrentDeviceId] = useState<string>("");
  const [revokingFamilyId, setRevokingFamilyId] = useState<string | null>(null);
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [revokeLoading, setRevokeLoading] = useState(false);

  useEffect(() => {
    if (open) {
      void loadSessions();
      fpPromise.load()
        .then(fp => fp.get())
        .then(result => { setCurrentDeviceId(result.visitorId); })
        .catch(console.error);
    } else {
      setRevokingFamilyId(null);
      setPassword("");
    }
  }, [open]);

  const loadSessions = async () => {
    try {
      setLoading(true);
      const data = await getSessions();
      setSessions(data);
    } catch {
      toast.error("Failed to load active sessions");
    } finally {
      setLoading(false);
    }
  };

  const handleRevokeClick = (familyId: string) => {
    setRevokingFamilyId(familyId);
    setPassword("");
  };

  const handleConfirmRevoke = async () => {
    if (!revokingFamilyId) return;
    if (!password) {
      toast.error("Please enter your password");
      return;
    }
    try {
      setRevokeLoading(true);
      await revokeSession(revokingFamilyId, password);
      toast.success("Session revoked successfully");
      setRevokingFamilyId(null);
      setPassword("");
      await loadSessions();
    } catch {
      toast.error("Failed to revoke session or invalid password");
    } finally {
      setRevokeLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px] max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <MonitorSmartphone className="h-5 w-5" />
            Active Devices
          </DialogTitle>
          <DialogDescription>
            Manage the devices that are currently logged into your account.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-auto mt-2">
          {revokingFamilyId ? (
            <div className="flex flex-col space-y-4 p-4 border rounded-lg bg-card m-1">
              <h3 className="font-medium">Confirm Revocation</h3>
              <p className="text-sm text-muted-foreground">
                Please enter your password to confirm you want to revoke access for this device.
              </p>
              <div className="space-y-2">
                <div className="relative">
                  <Input
                    type={showPassword ? "text" : "password"}
                    placeholder="Enter your password"
                    value={password}
                    onChange={(e) => { setPassword(e.target.value); }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        void handleConfirmRevoke();
                      }
                    }}
                    autoFocus
                    className="pr-10"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                    onClick={() => { setShowPassword(!showPassword); }}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <Eye className="h-4 w-4 text-muted-foreground" />
                    )}
                  </Button>
                </div>
              </div>
              <div className="flex justify-end space-x-2 pt-2">
                <Button variant="outline" onClick={() => { setRevokingFamilyId(null); }} disabled={revokeLoading}>
                  Cancel
                </Button>
                <Button onClick={() => { void handleConfirmRevoke(); }} disabled={revokeLoading}>
                  {revokeLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Confirm Revoke
                </Button>
              </div>
            </div>
          ) : loading ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          ) : (
            <div className="flex flex-col space-y-3 p-1">
              {sessions.map((s) => {
                const isCurrent = s.device_id === currentDeviceId;
                return (
                  <div 
                    key={s.family_id} 
                    className="flex items-center justify-between p-3 border rounded-lg bg-card shadow-sm"
                  >
                    <div className="flex flex-col gap-1.5">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm">
                          {getOSFromUserAgent(s.device_info)}
                        </span>
                        {isCurrent && (
                          <span className="text-[10px] bg-primary/10 text-primary px-2 py-0.5 rounded-full font-medium">
                            Current
                          </span>
                        )}
                      </div>
                      <span className="text-xs text-muted-foreground">
                        Last active: {new Date(s.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="ml-4 shrink-0">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => { handleRevokeClick(s.family_id); }}
                        className="h-8 w-8 text-red-600 hover:text-red-700 hover:bg-red-100 dark:hover:bg-red-900/30"
                        title="Revoke Session"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                );
              })}
              {sessions.length === 0 && (
                <div className="text-center py-8 text-muted-foreground border rounded-lg border-dashed">
                  No active sessions found.
                </div>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
