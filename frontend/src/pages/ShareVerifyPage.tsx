import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { checkSharePermission, verifySharePin } from "@/api/api-share";
import { Loader2, Shield, Fingerprint } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "sonner";
import VersionTag from "@/components/custom/versionTag";
import OtpInput from 'react-otp-input';
import { useTheme } from "@/components/theme-provider";
import { Logo } from "@/components/Logo";
import { ShareModeToggle } from "@/components/share/ShareModeToggle";

interface ApiError {
  response?: {
    status?: number;
    data?: {
      error?: string;
    };
  };
}

const otpInputClasses = [
  "w-10 h-12 sm:w-12 sm:h-14 text-center text-lg font-semibold",
  "border-2 rounded-xl bg-background transition-all duration-150",
  "focus:ring-2 focus:ring-primary/50 focus:border-primary focus:outline-none",
  "[&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]",
].join(" ");

export default function ShareVerifyPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [verifyingPin, setVerifyingPin] = useState(false);
  const [pin, setPin] = useState("");
  const [errorMsg, setErrorMsg] = useState("");
  const [serverName, setServerName] = useState("Go File Server");
  const { setTheme } = useTheme();

  useEffect(() => {
    fetch("/api/health")
      .then((res) => res.json())
      .then((data: { service_name?: string }) => {
        if (data.service_name) setServerName(data.service_name);
      })
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    if (!sessionStorage.getItem("share_theme_toggled")) {
      setTheme("system");
    }
    if (!id) {
      setErrorMsg("No share link ID provided.");
      setLoading(false);
      return;
    }

    const checkPermission = async () => {
      try {
        const rs = await checkSharePermission(id);
        sessionStorage.setItem("share_authority", rs.authority);
        void navigate(`/share/${id}/home`, { replace: true });
      } catch (err: unknown) {
        const error = err as ApiError;
        setLoading(false);
        if (
          error.response?.status === 404 ||
          error.response?.status === 410
        ) {
          setErrorMsg(
            error.response.data?.error ??
              "Link has expired or does not exist."
          );
        }
      }
    };

    void checkPermission();
  }, [id, navigate, setTheme]);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || pin.length !== 6) return;

    setVerifyingPin(true);
    setErrorMsg("");

    try {
      const rs = await verifySharePin(id, pin);
      sessionStorage.setItem("share_authority", rs.authority);
      toast.success("Verification successful");
      void navigate(`/share/${id}/home`, { replace: true });
    } catch (err: unknown) {
      const error = err as ApiError;
      setVerifyingPin(false);
      const msg = error.response?.data?.error ?? "Verification failed";
      setErrorMsg(msg);
      toast.error(msg);
    }
  };

  const isInvalidLink = !loading && errorMsg && !id;

  const titleContent = loading
    ? { title: "Verifying Access", subtitle: "Checking share link permissions..." }
    : isInvalidLink
      ? { title: "Invalid Link", subtitle: errorMsg }
      : { title: "Secure Share", subtitle: "Enter the 6-digit PIN to access the shared files." };

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
            This link is password-protected. Enter the PIN to access the shared files.
          </p>
          <div className="mt-12 flex gap-6 justify-center text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-primary" />
              <span>Encrypted</span>
            </div>
            <div className="flex items-center gap-2">
              <Fingerprint className="w-4 h-4 text-primary" />
              <span>Protected</span>
            </div>
          </div>
        </div>
      </div>

      {/* Right Panel — Form */}
      <div className="flex-1 flex flex-col items-center justify-center p-4 sm:p-8 md:p-12 relative">
        {/* Theme Toggle */}
        <div className="absolute top-4 right-4 z-10">
          <ShareModeToggle />
        </div>

        {/* Mobile gradient orbs */}
        <div className="lg:hidden absolute inset-0 overflow-hidden pointer-events-none">
          <div className="absolute -top-20 right-0 w-64 h-64 bg-primary/10 rounded-full blur-3xl translate-x-1/3" />
          <div className="absolute top-1/3 -left-10 w-48 h-48 bg-primary/10 rounded-full blur-3xl" />
          <div className="absolute -bottom-10 right-10 w-56 h-56 bg-primary/5 rounded-full blur-3xl" />
        </div>

        {/* Mobile branding */}
        <div className="lg:hidden relative w-full max-w-[360px] text-center mb-8">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 ring-1 ring-primary/20 mb-5">
            <Logo className="w-12 h-12 object-contain" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground mb-2">
            {serverName}
          </h1>
          <p className="text-sm text-muted-foreground leading-relaxed max-w-xs mx-auto mb-5">
            This link is password-protected. Enter the PIN to access the shared files.
          </p>
          <div className="flex gap-6 justify-center text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-primary" />
              <span>Encrypted</span>
            </div>
            <div className="flex items-center gap-2">
              <Fingerprint className="w-4 h-4 text-primary" />
              <span>Protected</span>
            </div>
          </div>
        </div>

        {/* Title / Subtitle */}
        <div className="w-full max-w-[360px] space-y-2 text-center mb-6 sm:mb-8 transition-all duration-200">
          <h2
            key={`title-${titleContent.title}`}
            className="text-xl sm:text-2xl font-semibold tracking-tight text-foreground animate-in fade-in slide-in-from-bottom-2 duration-200"
          >
            {titleContent.title}
          </h2>
          <p
            key={`sub-${titleContent.subtitle}`}
            className="text-base sm:text-sm text-muted-foreground animate-in fade-in slide-in-from-bottom-2 duration-200"
          >
            {titleContent.subtitle}
          </p>
        </div>

        {/* Card */}
        <Card className="w-full max-w-[360px] shadow-lg border-0 ring-1 ring-border/50 transition-all duration-200 py-0">
          <CardContent className="p-6">
            {loading ? (
              <div className="flex flex-col items-center gap-4 py-4">
                <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-primary/10">
                  <Loader2 className="h-6 w-6 animate-spin text-primary" />
                </div>
                <p className="text-sm text-muted-foreground">Please wait...</p>
              </div>
            ) : isInvalidLink ? (
              <div className="flex flex-col items-center gap-4 py-4">
                <p className="text-sm text-muted-foreground text-center">
                  This share link is no longer valid. Please request a new link.
                </p>
              </div>
            ) : (
              <form
                onSubmit={(e) => {
                  void handleVerify(e);
                }}
                className="flex flex-col items-center"
              >
                <div className="flex justify-center w-full">
                  <OtpInput
                    value={pin}
                    onChange={setPin}
                    numInputs={6}
                    shouldAutoFocus={true}
                    renderSeparator={
                      <span className="mx-0.5 sm:mx-1 text-muted-foreground/30 select-none">
                        &bull;
                      </span>
                    }
                    renderInput={(props) => (
                      <input
                        {...props}
                        type="number"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        disabled={verifyingPin}
                        className={otpInputClasses}
                        style={{ width: "2.5rem" }}
                      />
                    )}
                  />
                </div>

                {errorMsg && (
                  <p className="text-sm text-destructive text-center font-medium mt-4">
                    {errorMsg}
                  </p>
                )}

                <Button
                  type="submit"
                  className="w-full h-11 rounded-xl font-medium mt-8"
                  disabled={pin.length !== 6 || verifyingPin}
                >
                  {verifyingPin ? (
                    <span className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      Verifying...
                    </span>
                  ) : (
                    "Verify"
                  )}
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
}
