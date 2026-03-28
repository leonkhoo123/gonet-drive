import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { checkSharePermission, verifySharePin } from "@/api/api-share";
import { Loader2, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { toast } from "sonner";
import DefaultLayout from "@/layouts/DefaultLayout";
import VersionTag from "@/components/custom/versionTag";
import OtpInput from 'react-otp-input';
import { useTheme } from "@/components/theme-provider";

interface ApiError {
  response?: {
    status?: number;
    data?: {
      error?: string;
    }
  }
}

export default function ShareVerifyPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [verifyingPin, setVerifyingPin] = useState(false);
  const [pin, setPin] = useState("");
  const [errorMsg, setErrorMsg] = useState("");
  const { setTheme } = useTheme();

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
        // Valid token already exists
        sessionStorage.setItem("share_authority", rs.authority);
        void navigate(`/share/${id}/home`, { replace: true });
      } catch (err: unknown) {
        const error = err as ApiError;
        // Expected if no cookie or expired, just show the PIN form
        setLoading(false);
        if (error.response?.status === 404 || error.response?.status === 410) {
           setErrorMsg(error.response.data?.error ?? "Link has expired or does not exist.");
        }
      }
    };

    void checkPermission();
  }, [id, navigate, setTheme]);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !pin || pin.length < 6) return;

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

  return (
    <DefaultLayout>
      <div className="flex flex-1 flex-col items-center justify-center p-4">
        {loading ? (
          <div className="flex flex-col items-center text-muted-foreground">
            <Loader2 className="h-10 w-10 animate-spin text-primary mb-4" />
            <p className="text-lg">Verifying access...</p>
          </div>
        ) : errorMsg && !id ? (
          <Card className="w-full max-w-md shadow-lg border-red-200 dark:border-red-900">
             <CardHeader className="text-center">
                <CardTitle className="text-red-500">Invalid Link</CardTitle>
                <CardDescription>{errorMsg}</CardDescription>
             </CardHeader>
          </Card>
        ) : (
          <Card className="w-full max-w-md shadow-lg">
            <CardHeader className="text-center pb-4">
              <div className="mx-auto w-12 h-12 bg-primary/10 text-primary rounded-full flex items-center justify-center mb-4">
                <ShieldCheck className="h-6 w-6" />
              </div>
              <CardTitle className="text-2xl font-bold">Secure Share</CardTitle>
              <CardDescription className="text-base mt-2">
                This link is protected. Please enter the 6-digit PIN to access the shared files.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={(e) => { void handleVerify(e); }} className="space-y-6 flex flex-col items-center">
                <div className="space-y-2 w-full flex flex-col items-center">
                  <div className="flex justify-center w-full">
                    <OtpInput
                      value={pin}
                      onChange={setPin}
                      numInputs={6}
                      shouldAutoFocus={true}
                      renderSeparator={<span className="mx-1">-</span>}
                      renderInput={(props) => (
                        <input
                          {...props}
                          type="password"
                          inputMode="numeric"
                          pattern="[0-9]*"
                          disabled={verifyingPin}
                          className="w-12 h-12 text-center text-lg border rounded-md focus:ring-2 focus:ring-primary focus:outline-none dark:bg-zinc-950 dark:border-zinc-800 [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]"
                          style={{ width: "3rem" }}
                        />
                      )}
                    />
                  </div>
                  {errorMsg && (
                    <p className="text-sm text-red-500 text-center font-medium mt-2">{errorMsg}</p>
                  )}
                </div>
                <Button 
                  type="submit" 
                  className="w-full h-11 text-base font-semibold" 
                  disabled={pin.length < 6 || verifyingPin}
                >
                  {verifyingPin ? (
                    <>
                      <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                      Authenticating...
                    </>
                  ) : (
                    "Verify PIN"
                  )}
                </Button>
              </form>
            </CardContent>
            <CardFooter className="justify-center pt-2 pb-6">
              <p className="text-xs text-muted-foreground">
                Protected by Go File Server
              </p>
            </CardFooter>
          </Card>
        )}
      </div>
      <VersionTag />
    </DefaultLayout>
  );
}