import { useState, useEffect, Suspense, useRef } from 'react';
import { Outlet, useNavigate } from 'react-router-dom';
import { checkAuthStatus } from '@/api/api-auth';
import { VerificationScreen } from './VerificationScreen';

const MIN_VERIFY_MS = 400;

export function AuthGate() {
  const [status, setStatus] = useState<'verifying' | 'authenticated'>('verifying');
  const navigate = useNavigate();
  const navigatingRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    navigatingRef.current = false;
    const startTime = Date.now();

    void checkAuthStatus()
      .then(async () => {
        const elapsed = Date.now() - startTime;
        if (elapsed < MIN_VERIFY_MS) {
          await new Promise(resolve => setTimeout(resolve, MIN_VERIFY_MS - elapsed));
        }
        if (!cancelled) setStatus('authenticated');
      })
      .catch(async () => {
        const elapsed = Date.now() - startTime;
        if (elapsed < MIN_VERIFY_MS) {
          await new Promise(resolve => setTimeout(resolve, MIN_VERIFY_MS - elapsed));
        }
        if (!cancelled && !navigatingRef.current) {
          navigatingRef.current = true;
          void navigate('/login', { replace: true });
        }
      });

    return () => { cancelled = true; };
  }, [navigate]);

  if (status === 'verifying') {
    return <VerificationScreen />;
  }

  return (
    <Suspense fallback={<VerificationScreen subtitle="Loading..." />}>
      <Outlet />
    </Suspense>
  );
}
