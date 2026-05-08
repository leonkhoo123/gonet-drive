import { useState, useEffect, useRef, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
} from '@/components/ui/card';
import { login, verifyMfa, setupMfa, enableMfa, checkAuthStatus } from '@/api/api-auth';
import { useNavigate, useLocation } from 'react-router-dom';
import { toast } from 'sonner';
import VersionTag from '@/components/custom/versionTag';
import OtpInput from 'react-otp-input';
import { QRCodeSVG } from 'qrcode.react';
import axios from 'axios';
import { Eye, EyeOff, ArrowRight, Shield, Fingerprint, User, Lock, Loader2 } from 'lucide-react';
import { Logo } from '@/components/Logo';

type Step = 'login' | 'mfa' | 'mfaSetup';

const otpInputClasses = [
  "w-10 h-12 sm:w-12 sm:h-14 text-center text-lg font-semibold",
  "border-2 rounded-xl bg-background transition-all duration-150",
  "focus:ring-2 focus:ring-primary/50 focus:border-primary focus:outline-none",
  "[&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]",
].join(' ');

const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [mfaCode, setMfaCode] = useState('');
  const [qrUrl, setQrUrl] = useState('');
  const [setupSecret, setSetupSecret] = useState('');
  const [serverName, setServerName] = useState('GoNet Drive');
  const [isLoading, setIsLoading] = useState(false);
  const [step, setStep] = useState<Step>('login');
  const [animating, setAnimating] = useState(false);

  const passwordRef = useRef<HTMLInputElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const loginFormRef = useRef<HTMLFormElement>(null);
  const mfaFormRef = useRef<HTMLFormElement>(null);
  const mfaSetupFormRef = useRef<HTMLFormElement>(null);

  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    fetch('/api/health')
      .then((res) => res.json())
      .then((data: { service_name?: string }) => {
        if (data.service_name) setServerName(data.service_name);
      })
      .catch(() => undefined);
  }, []);

  const transitionTo = useCallback((nextStep: Step) => {
    setAnimating(true);
    setTimeout(() => {
      setStep(nextStep);
      setAnimating(false);
    }, 180);
  }, []);

  const resetToLogin = useCallback(() => {
    setMfaCode('');
    transitionTo('login');
  }, [transitionTo]);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    if (params.get('mfa_setup_required') === 'true') {
      transitionTo('mfaSetup');
      setMfaCode('');
      setupMfa().then(setupRes => {
        setQrUrl(setupRes.url);
        setSetupSecret(setupRes.secret);
      }).catch(() => {
        toast.error("Failed to load MFA setup details");
      });
    } else {
      checkAuthStatus().then(() => {
        void navigate("/home");
      }).catch(() => {
        // Not logged in, stay on login page
      });
    }
  }, [location.search, navigate, transitionTo]);

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    try {
      const res = await login(username, password);
      if (res.mfa_required) {
        setMfaCode('');
        transitionTo('mfa');
        toast.info("Enter your 2FA code to continue.");
      } else if (res.mfa_setup_required) {
        setMfaCode('');
        transitionTo('mfaSetup');
        const setupRes = await setupMfa();
        setQrUrl(setupRes.url);
        setSetupSecret(setupRes.secret);
        toast.info("Please set up Two-Factor Authentication.");
      } else {
        toast.success("Welcome");
        void navigate("/home");
      }
    } catch (err: unknown) {
      if (axios.isAxiosError(err)) {
        toast.error((err.response?.data as { error?: string } | undefined)?.error ?? "Login Failed");
      } else {
        toast.error("Login Failed");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleMfaSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (mfaCode.length !== 6) return;
    setIsLoading(true);
    try {
      await verifyMfa(mfaCode);
      toast.success("Welcome");
      void navigate("/home");
    } catch (err: unknown) {
      if (axios.isAxiosError(err)) {
        toast.error((err.response?.data as { error?: string } | undefined)?.error ?? "Invalid MFA Code");
      } else {
        toast.error("Invalid MFA Code");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleMfaSetupSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (mfaCode.length !== 6) return;
    setIsLoading(true);
    try {
      await enableMfa(mfaCode);
      toast.success("MFA Setup Successful! Welcome");
      void navigate("/home");
    } catch (err: unknown) {
      if (axios.isAxiosError(err)) {
        toast.error((err.response?.data as { error?: string } | undefined)?.error ?? "Invalid MFA Code for setup");
      } else {
        toast.error("Invalid MFA Code for setup");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleUsernameKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && passwordRef.current) {
      e.preventDefault();
      passwordRef.current.focus();
    }
  };

  const stepTitles: Record<Step, { title: string; subtitle: string }> = {
    login: {
      title: 'Welcome back',
      subtitle: '',
    },
    mfa: {
      title: 'Two-Factor Authentication',
      subtitle: 'Enter the 6-digit code from your authenticator app.',
    },
    mfaSetup: {
      title: 'Set Up Two-Factor Auth',
      subtitle: 'Scan the QR code with your authenticator app, then enter the code.',
    },
  };

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
            Your personal cloud storage. Secure, fast, and always within reach.
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
        {/* Mobile-only gradient orbs behind content */}
        <div className="lg:hidden absolute inset-0 overflow-hidden pointer-events-none">
          <div className="absolute -top-20 right-0 w-64 h-64 bg-primary/10 rounded-full blur-3xl translate-x-1/3" />
          <div className="absolute top-1/3 -left-10 w-48 h-48 bg-primary/10 rounded-full blur-3xl" />
          <div className="absolute -bottom-10 right-10 w-56 h-56 bg-primary/5 rounded-full blur-3xl" />
        </div>

        {/* Mobile branding (hidden on desktop) */}
        <div className="lg:hidden relative w-full max-w-[360px] text-center mb-8">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 ring-1 ring-primary/20 mb-5">
            <Logo className="w-12 h-12 object-contain" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground mb-2">
            {serverName}
          </h1>
          <p className="text-sm text-muted-foreground leading-relaxed max-w-xs mx-auto mb-5">
            Your personal cloud storage. Secure, fast, and always within reach.
          </p>
          <div className="flex gap-6 justify-center text-sm text-muted-foreground">
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

        <div
          ref={cardRef}
          className="w-full max-w-[360px] space-y-2 text-center mb-6 sm:mb-8 transition-all duration-200"
        >
          <h2
            key={`title-${step}`}
            className="text-xl sm:text-2xl font-semibold tracking-tight text-foreground animate-in fade-in slide-in-from-bottom-2 duration-200"
          >
            {stepTitles[step].title}
          </h2>
          {stepTitles[step].subtitle && (
            <p
              key={`sub-${step}`}
              className="text-base sm:text-sm text-muted-foreground animate-in fade-in slide-in-from-bottom-2 duration-200"
            >
              {stepTitles[step].subtitle}
            </p>
          )}
        </div>

        <Card className={`w-full max-w-[360px] shadow-lg border-0 ring-1 ring-border/50 transition-all duration-200 py-0 ${animating ? 'opacity-0 scale-[0.98]' : 'opacity-100 scale-100'}`}>
          <CardContent className="p-6">

            {/* ---- LOGIN FORM ---- */}
            {step === 'login' && (
              <form ref={loginFormRef} onSubmit={(e) => { void handleLoginSubmit(e); }} className="space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="username" className="text-sm font-medium">Username</Label>
                  <div className="relative">
                    <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      id="username"
                      type="text"
                      placeholder="Enter your username"
                      required
                      autoFocus
                      value={username}
                      onChange={(e) => { setUsername(e.target.value); }}
                      onKeyDown={handleUsernameKeyDown}
                      className="pl-10 h-11 rounded-xl"
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
                      placeholder="Enter your password"
                      required
                      value={password}
                      onChange={(e) => { setPassword(e.target.value); }}
                      className="pl-10 pr-10 h-11 rounded-xl"
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
                <Button type="submit" className="w-full h-11 rounded-xl font-medium" disabled={isLoading}>
                  {isLoading ? (
                    <span className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Signing in...
                    </span>
                  ) : (
                    <span className="flex items-center gap-2">
                      Sign In <ArrowRight className="h-4 w-4" />
                    </span>
                  )}
                </Button>
              </form>
            )}

            {/* ---- MFA VERIFY FORM ---- */}
            {step === 'mfa' && (
              <form ref={mfaFormRef} onSubmit={(e) => { void handleMfaSubmit(e); }} className="flex flex-col items-center">

                <div className="flex justify-center w-full">
                  <OtpInput
                    value={mfaCode}
                    onChange={setMfaCode}
                    numInputs={6}
                    shouldAutoFocus
                    renderSeparator={<span className="mx-0.5 sm:mx-1 text-muted-foreground/30 select-none">&bull;</span>}
                    renderInput={(props) => (
                      <input
                        {...props}
                        type="number"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        className={otpInputClasses}
                        style={{ width: "2.5rem" }}
                      />
                    )}
                  />
                </div>

                <Button
                  type="submit"
                  className="w-full h-11 rounded-xl font-medium mt-8"
                  disabled={mfaCode.length !== 6 || isLoading}
                >
                  {isLoading ? (
                    <span className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Verifying...
                    </span>
                  ) : (
                    'Verify'
                  )}
                </Button>
                <Button type="button" variant="ghost" className="w-full rounded-xl mt-2" onClick={resetToLogin}>
                  Back
                </Button>
              </form>
            )}

            {/* ---- MFA SETUP FORM ---- */}
            {step === 'mfaSetup' && (
              <form ref={mfaSetupFormRef} onSubmit={(e) => { void handleMfaSetupSubmit(e); }} className="flex flex-col items-center">

                <div className="text-center text-base text-muted-foreground mb-4">
                  Scan with an authenticator app like Google Authenticator or Authy.
                </div>

                {qrUrl && (
                  <div className="bg-white p-3 rounded-xl ring-1 ring-border/50 flex justify-center mb-4">
                    <QRCodeSVG value={qrUrl} size={150} className="sm:w-[180px] sm:h-[180px]" />
                  </div>
                )}

                <div className="text-center text-xs font-mono bg-muted p-2.5 rounded-xl w-full break-all mb-5">
                  {setupSecret}
                </div>

                <p className="text-base text-muted-foreground mb-4">Enter the 6-digit code shown in your app:</p>

                <div className="flex justify-center w-full">
                  <OtpInput
                    value={mfaCode}
                    onChange={setMfaCode}
                    numInputs={6}
                    shouldAutoFocus
                    renderSeparator={<span className="mx-0.5 sm:mx-1 text-muted-foreground/30 select-none">&bull;</span>}
                    renderInput={(props) => (
                      <input
                        {...props}
                        type="number"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        className={otpInputClasses}
                        style={{ width: "2.5rem" }}
                      />
                    )}
                  />
                </div>

                <Button
                  type="submit"
                  className="w-full h-11 rounded-xl font-medium mt-6"
                  disabled={mfaCode.length !== 6 || isLoading}
                >
                  {isLoading ? (
                    <span className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Verifying...
                    </span>
                  ) : (
                    'Verify & Enable'
                  )}
                </Button>
                <Button type="button" variant="ghost" className="w-full rounded-xl mt-2" onClick={resetToLogin}>
                  Back
                </Button>
              </form>
            )}

          </CardContent>
        </Card>

        <div className="mt-6">
          <VersionTag />
        </div>
      </div>
    </div>
  );
};

export default LoginPage;