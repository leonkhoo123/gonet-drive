import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button'; // shadcn/ui Button
import { Input } from '@/components/ui/input';   // shadcn/ui Input
import { Label } from '@/components/ui/label';   // shadcn/ui Label
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'; // shadcn/ui Card components
import { login, verifyMfa, setupMfa, enableMfa, checkAuthStatus } from '@/api/api-auth';
import { useNavigate, useLocation } from 'react-router-dom';
import { toast } from 'sonner';
import VersionTag from '@/components/custom/versionTag';
import OtpInput from 'react-otp-input';
import { QRCodeSVG } from 'qrcode.react';
import axios from 'axios';
import { Eye, EyeOff } from 'lucide-react';

const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [mfaRequired, setMfaRequired] = useState(false);
  const [mfaSetupRequired, setMfaSetupRequired] = useState(false);
  const [mfaCode, setMfaCode] = useState('');
  const [qrUrl, setQrUrl] = useState('');
  const [setupSecret, setSetupSecret] = useState('');
  
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    if (params.get('mfa_setup_required') === 'true') {
      setMfaSetupRequired(true);
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
  }, [location.search, navigate]);

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await login(username, password);
      if (res.mfa_required) {
        setMfaRequired(true);
        toast.info("MFA required. Please enter your code.");
      } else if (res.mfa_setup_required) {
        setMfaSetupRequired(true);
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
    }
  };

  const handleMfaSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
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
    }
  };

  const handleMfaSetupSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
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
    }
  };

  return (
      <div className="flex justify-center items-center p-4 min-h-[calc(100dvh-4rem)]">
        <Card className="w-full max-w-md shadow-xl transition-all duration-300">
          <CardHeader className="text-center">
            <CardTitle className="text-3xl font-bold tracking-tight">
              {mfaRequired ? 'Two-Factor Authentication' : mfaSetupRequired ? 'Set Up 2FA' : 'Login'}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!mfaRequired && !mfaSetupRequired ? (
              <form onSubmit={handleLoginSubmit} className="space-y-6">
                <div className="space-y-2">
                  <Label htmlFor="username">Username</Label>
                  <Input
                    id="username"
                    type="text"
                    placeholder="Username"
                    required
                    value={username}
                    onChange={(e) => { setUsername(e.target.value); }}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password">Password</Label>
                  <div className="relative">
                    <Input
                      id="password"
                      type={showPassword ? "text" : "password"}
                      placeholder="Password"
                      required
                      value={password}
                      onChange={(e) => { setPassword(e.target.value); }}
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
                <Button type="submit" className="w-full">
                  Sign In
                </Button>
              </form>
            ) : mfaSetupRequired ? (
              <form onSubmit={handleMfaSetupSubmit} className="space-y-6 flex flex-col items-center">
                <div className="text-center text-sm text-muted-foreground mb-4">
                  Scan this QR code with your authenticator app (like Google Authenticator or Authy).
                </div>
                {qrUrl && (
                  <div className="bg-white p-4 rounded-md mb-4 flex justify-center">
                    <QRCodeSVG value={qrUrl} size={200} />
                  </div>
                )}
                <div className="text-center text-xs font-mono bg-zinc-100 dark:bg-zinc-900 p-2 rounded w-full break-all mb-4">
                  {setupSecret}
                </div>
                <div className="flex justify-center w-full">
                  <OtpInput
                    value={mfaCode}
                    onChange={setMfaCode}
                    numInputs={6}
                    renderSeparator={<span className="mx-1">-</span>}
                    renderInput={(props) => (
                      <input
                        {...props}
                        type="number"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        className="w-12 h-12 text-center text-lg border rounded-md focus:ring-2 focus:ring-primary focus:outline-none dark:bg-zinc-950 dark:border-zinc-800 [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]"
                        style={{ width: "3rem" }}
                      />
                    )}
                  />
                </div>
                <Button type="submit" className="w-full mt-6" disabled={mfaCode.length !== 6}>
                  Verify & Enable
                </Button>
                <Button type="button" variant="ghost" className="w-full mt-2" onClick={() => { setMfaSetupRequired(false); setMfaCode(''); void navigate("/login"); }}>
                  Cancel
                </Button>
              </form>
            ) : (
              <form onSubmit={handleMfaSubmit} className="space-y-6 flex flex-col items-center">
                <div className="text-center text-sm text-muted-foreground mb-4">
                  Enter the 6-digit code from your authenticator app.
                </div>
                <div className="flex justify-center w-full">
                  <OtpInput
                    value={mfaCode}
                    onChange={setMfaCode}
                    numInputs={6}
                    renderSeparator={<span className="mx-1">-</span>}
                    renderInput={(props) => (
                      <input
                        {...props}
                        type="number"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        className="w-12 h-12 text-center text-lg border rounded-md focus:ring-2 focus:ring-primary focus:outline-none dark:bg-zinc-950 dark:border-zinc-800 [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]"
                        style={{ width: "3rem" }}
                      />
                    )}
                  />
                </div>
                <Button type="submit" className="w-full mt-6" disabled={mfaCode.length !== 6}>
                  Verify
                </Button>
                <Button type="button" variant="ghost" className="w-full mt-2" onClick={() => { setMfaRequired(false); setMfaCode(''); void navigate("/login"); }}>
                  Back to Login
                </Button>
              </form>
            )}
          </CardContent>
          <VersionTag />
        </Card>
      </div>
  );
};

export default LoginPage;