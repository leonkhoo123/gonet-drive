import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { Logo } from '@/components/Logo';

interface VerificationScreenProps {
  subtitle?: string;
}

export function VerificationScreen({
  subtitle = "Verifying your session...",
}: VerificationScreenProps) {
  const [serverName, setServerName] = useState('GoNet Drive');

  useEffect(() => {
    void fetch('/api/health')
      .then(res => res.json() as Promise<{ service_name?: string }>)
      .then(data => {
        if (data.service_name) setServerName(data.service_name);
      })
      .catch(() => { /* server name fallback is already set */ });
  }, []);

  return (
    <div className="flex min-h-dvh bg-gradient-to-br from-primary/5 via-background to-primary/10">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 right-0 w-96 h-96 bg-primary/10 rounded-full blur-3xl translate-x-1/2 -translate-y-1/2" />
        <div className="absolute bottom-0 left-0 w-72 h-72 bg-primary/10 rounded-full blur-3xl -translate-x-1/2 translate-y-1/2" />
        <div className="absolute top-1/3 left-1/2 -translate-x-1/2 w-64 h-64 bg-primary/5 rounded-full blur-3xl" />
      </div>

      <div className="relative z-10 flex-1 flex flex-col items-center justify-center p-4">
        <div className="text-center max-w-md">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 ring-1 ring-primary/20 mb-8 animate-in fade-in zoom-in duration-500">
            <Logo className="w-12 h-12 object-contain" />
          </div>

          <h1 className="text-3xl sm:text-4xl font-bold tracking-tight text-foreground mb-6 animate-in fade-in slide-in-from-bottom-2 duration-500">
            {serverName}
          </h1>

          <div className="flex items-center justify-center gap-3 text-muted-foreground animate-in fade-in slide-in-from-bottom-4 duration-700">
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
            <span className="text-lg">{subtitle}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
