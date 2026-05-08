import { useState, useEffect, Suspense } from 'react';
import { Outlet } from 'react-router-dom';
import { checkAuthStatus } from '@/api/api-auth';
import { VerificationScreen } from './VerificationScreen';

export function AuthGate() {
  const [status, setStatus] = useState<'verifying' | 'authenticated'>('verifying');

  useEffect(() => {
    let cancelled = false;

    void checkAuthStatus()
      .then(() => {
        if (!cancelled) setStatus('authenticated');
      })
      .catch(() => {
        if (!cancelled) {
          window.location.href = '/login';
        }
      });

    return () => { cancelled = true; };
  }, []);

  if (status === 'verifying') {
    return <VerificationScreen />;
  }

  return (
    <Suspense fallback={<VerificationScreen subtitle="Loading..." />}>
      <Outlet />
    </Suspense>
  );
}
