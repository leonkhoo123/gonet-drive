import type { RefObject } from 'react';
import { Button } from '@/components/ui/button';
import { QRCodeSVG } from 'qrcode.react';
import { Eye, EyeOff, KeyRound, Shield, Copy, Check, Loader2 } from 'lucide-react';
import OtpCodeInput from '@/pages/login/OtpCodeInput';

interface MfaSetupPanelProps {
  formRef: RefObject<HTMLFormElement | null>;
  mfaCode: string;
  setMfaCode: (value: string) => void;
  qrUrl: string;
  setupSecret: string;
  showSecret: boolean;
  onToggleSecret: () => void;
  secretCopied: boolean;
  onCopySecret: () => void;
  isLoading: boolean;
  onSubmit: (e: React.FormEvent) => void;
  onReset: () => void;
  showRecoveryCodes: boolean;
  recoveryCodes: string[];
  recoveryCodesCopied: boolean;
  onCopyRecoveryCodes: () => void;
  onDone: () => void;
}

const MfaSetupPanel: React.FC<MfaSetupPanelProps> = ({
  formRef,
  mfaCode,
  setMfaCode,
  qrUrl,
  setupSecret,
  showSecret,
  onToggleSecret,
  secretCopied,
  onCopySecret,
  isLoading,
  onSubmit,
  onReset,
  showRecoveryCodes,
  recoveryCodes,
  recoveryCodesCopied,
  onCopyRecoveryCodes,
  onDone,
}) => {
  if (showRecoveryCodes) {
    return (
      <div className="flex flex-col items-center">
        <div className="w-full bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 rounded-xl p-3 mb-4 text-sm text-amber-800 dark:text-amber-200">
          Store this code securely. If you lose access to your authenticator app, you can use it once to recover your account.
        </div>
        <div className="w-full bg-muted font-mono text-sm py-3 px-4 rounded-xl text-center tracking-widest select-all cursor-pointer hover:bg-muted/80 transition-colors mb-4">
          {recoveryCodes[0]}
        </div>
        <button
          type="button"
          onClick={onCopyRecoveryCodes}
          className="w-full flex items-center justify-center gap-2 py-2.5 mb-4 rounded-xl bg-muted/60 hover:bg-muted ring-1 ring-border/50 text-sm text-muted-foreground hover:text-foreground transition-all duration-150"
        >
          {recoveryCodesCopied ? (
            <>
              <Check className="w-4 h-4 text-green-500" />
              <span>Copied to clipboard</span>
            </>
          ) : (
            <>
              <Copy className="w-4 h-4" />
              <span>Copy recovery code</span>
            </>
          )}
        </button>
        <Button
          type="button"
          className="w-full h-11 rounded-xl font-medium"
          onClick={onDone}
        >
          I have saved my recovery code
        </Button>
        <Button type="button" variant="ghost" className="w-full rounded-xl mt-2" onClick={onReset}>
          Back
        </Button>
      </div>
    );
  }

  return (
    <form ref={formRef} onSubmit={onSubmit} className="flex flex-col items-center">
      <div className="text-center text-base text-muted-foreground mb-4">
        Scan with an authenticator app like Google Authenticator or Authy.
      </div>

      {qrUrl && (
        <div className="bg-white p-3 rounded-xl ring-1 ring-border/50 flex justify-center mb-4">
          <QRCodeSVG value={qrUrl} size={150} className="sm:w-[180px] sm:h-[180px]" />
        </div>
      )}

      <div className="w-full mb-5">
        <button
          type="button"
          onClick={onToggleSecret}
          className="w-full flex items-center justify-between px-3.5 py-2.5 rounded-xl bg-muted/60 hover:bg-muted ring-1 ring-border/50 transition-all duration-150 group"
        >
          <div className="flex items-center gap-2.5">
            <KeyRound className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
            <span className="text-sm text-muted-foreground group-hover:text-foreground transition-colors">
              {showSecret ? 'Hide secret key' : 'Show secret key'}
            </span>
          </div>
          <Eye className={`w-4 h-4 text-muted-foreground transition-all duration-200 ${showSecret ? 'hidden' : 'block'}`} />
          <EyeOff className={`w-4 h-4 text-muted-foreground transition-all duration-200 ${showSecret ? 'block' : 'hidden'}`} />
        </button>
        {showSecret && (
          <div className="mt-2 px-3.5 py-3 rounded-xl bg-muted/40 ring-1 ring-border/50 animate-in fade-in slide-in-from-top-1 duration-150">
            <div className="flex items-center justify-between gap-2">
              <code className="text-xs font-mono text-foreground/80 break-all select-all leading-relaxed flex-1">{setupSecret}</code>
              <button
                type="button"
                onClick={onCopySecret}
                className="shrink-0 p-1.5 rounded-lg hover:bg-background text-muted-foreground hover:text-foreground transition-all duration-150"
                title="Copy to clipboard"
              >
                {secretCopied ? (
                  <Check className="w-4 h-4 text-green-500" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </button>
            </div>
            <p className="text-[11px] text-muted-foreground/70 mt-2 flex items-center gap-1.5">
              <Shield className="w-3 h-3 shrink-0" />
              Never share this key with anyone
            </p>
          </div>
        )}
      </div>

      <p className="text-base text-muted-foreground mb-4">Enter the 6-digit code shown in your app:</p>

      <OtpCodeInput value={mfaCode} onChange={setMfaCode} />

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
      <Button type="button" variant="ghost" className="w-full rounded-xl mt-2" onClick={onReset}>
        Back
      </Button>
    </form>
  );
};

export default MfaSetupPanel;
