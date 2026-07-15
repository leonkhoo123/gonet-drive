import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';
import { setupAdmin, getSetupStatus, checkAuthStatus } from '@/api/api-auth';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import axios from 'axios';
import { Eye, EyeOff, ArrowRight, Shield, Fingerprint, User, Lock, Loader2 } from 'lucide-react';
import { Logo } from '@/components/Logo';
import VersionTag from '@/components/custom/versionTag';

const SetupPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isChecking, setIsChecking] = useState(true);
  const [serverName] = useState('GoNet Drive');
  const passwordRef = useRef<HTMLInputElement>(null);
  const confirmPasswordRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    let cancelled = false;
    getSetupStatus()
      .then(async (res) => {
        if (res.setup_required) {
          if (!cancelled) setIsChecking(false);
          return;
        }
        try {
          await checkAuthStatus();
          if (!cancelled) void navigate('/home', { replace: true });
        } catch {
          if (!cancelled) void navigate('/login', { replace: true });
        }
      })
      .catch(() => { if (!cancelled) setIsChecking(false); });
    return () => { cancelled = true; };
  }, [navigate]);

  const handleUsernameKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && passwordRef.current) {
      e.preventDefault();
      passwordRef.current.focus();
    }
  };

  const handlePasswordKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && confirmPasswordRef.current) {
      e.preventDefault();
      confirmPasswordRef.current.focus();
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (password !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }

    setIsLoading(true);
    try {
      await setupAdmin(username, password);
      toast.success("Admin account created! Please sign in.");
      void navigate('/login');
    } catch (err: unknown) {
      if (axios.isAxiosError(err)) {
        toast.error((err.response?.data as { error?: string } | undefined)?.error ?? "Setup failed");
      } else {
        toast.error("Setup failed");
      }
    } finally {
      setIsLoading(false);
    }
  };

  if (isChecking) {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-gradient-to-br from-primary/5 via-background to-primary/10">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh bg-gradient-to-br from-primary/5 via-background to-primary/10 lg:from-background lg:via-background lg:to-primary/5">
      {/* Left Panel — Branding (desktop) */}
      <div className="hidden lg:flex lg:w-1/2 relative overflow-hidden items-center justify-center bg-gradient-to-br from-primary/10 via-primary/5 to-background">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/20 via-transparent to-transparent" />
        <div className="absolute top-0 right-0 w-96 h-96 bg-primary/10 rounded-full blur-3xl translate-x-1/2 -translate-y-1/2" />
        <div className="absolute bottom-0 left-0 w-72 h-72 bg-primary/10 rounded-full blur-3xl -translate-x-1/2 translate-y-1/2" />

        <div className="relative z-10 text-center px-12 max-w-md">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 ring-1 ring-primary/20 mb-8">
            <Logo className="w-12 h-12 object-contain" />
          </div>
          <h1 className="text-4xl font-bold tracking-tight text-foreground mb-4">
            {serverName}
          </h1>
          <p className="text-lg text-muted-foreground leading-relaxed">
            Create your administrator account to get started.
          </p>
          <div className="mt-12 flex gap-6 justify-center text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-primary" />
              <span>Encrypted</span>
            </div>
            <div className="flex items-center gap-2">
              <Fingerprint className="w-4 h-4 text-primary" />
              <span>2FA Ready</span>
            </div>
          </div>
        </div>
      </div>

      {/* Right Panel — Form */}
      <div className="flex-1 flex flex-col items-center justify-center p-4 sm:p-8 md:p-12 relative">
        {/* Mobile branding */}
        <div className="lg:hidden relative w-full max-w-[360px] text-center mb-8">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 ring-1 ring-primary/20 mb-5">
            <Logo className="w-12 h-12 object-contain" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground mb-2">
            {serverName}
          </h1>
          <p className="text-sm text-muted-foreground leading-relaxed max-w-xs mx-auto">
            Create your administrator account to get started.
          </p>
        </div>

        <div className="w-full max-w-[360px] space-y-2 text-center mb-6 sm:mb-8">
          <h2 className="text-xl sm:text-2xl font-semibold tracking-tight text-foreground">
            Initial Setup
          </h2>
          <p className="text-base sm:text-sm text-muted-foreground">
            Create an administrator account for your cloud storage.
          </p>
        </div>

        <Card className="w-full max-w-[360px] shadow-lg border-0 ring-1 ring-border/50 py-0">
          <CardContent className="p-6">
            <form onSubmit={(e) => { void handleSubmit(e); }} className="space-y-5">
              <div className="space-y-2">
                <Label htmlFor="username" className="text-sm font-medium">Username</Label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="username"
                    type="text"
                    placeholder="Choose a username (min 6 chars)"
                    required
                    autoFocus
                    value={username}
                    onChange={(e) => { setUsername(e.target.value); }}
                    onKeyDown={handleUsernameKeyDown}
                    className="pl-10 h-11 rounded-xl"
                    minLength={6}
                    maxLength={32}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="password" className="text-sm font-medium">Password</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="password"
                    ref={passwordRef}
                    type={showPassword ? "text" : "password"}
                    placeholder="Choose a password (min 8 chars)"
                    required
                    value={password}
                    onChange={(e) => { setPassword(e.target.value); }}
                    onKeyDown={handlePasswordKeyDown}
                    className="pl-10 pr-10 h-11 rounded-xl"
                    minLength={8}
                    maxLength={72}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-0 top-0 h-full px-3 hover:bg-transparent rounded-xl"
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
              <div className="space-y-2">
                <Label htmlFor="confirm-password" className="text-sm font-medium">Confirm Password</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="confirm-password"
                    ref={confirmPasswordRef}
                    type={showPassword ? "text" : "password"}
                    placeholder="Confirm your password"
                    required
                    value={confirmPassword}
                    onChange={(e) => { setConfirmPassword(e.target.value); }}
                    className="pl-10 h-11 rounded-xl"
                    minLength={8}
                    maxLength={72}
                  />
                </div>
              </div>
              <Button type="submit" className="w-full h-11 rounded-xl font-medium" disabled={isLoading}>
                {isLoading ? (
                  <span className="flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Creating account...
                  </span>
                ) : (
                  <span className="flex items-center gap-2">
                    Create Admin Account <ArrowRight className="h-4 w-4" />
                  </span>
                )}
              </Button>
            </form>
          </CardContent>
        </Card>

        <div className="mt-6">
          <VersionTag />
        </div>
      </div>
    </div>
  );
};

export default SetupPage;
