import './App.css'
import { ThemeProvider } from './components/theme-provider'
import { Route, Routes, useNavigate } from "react-router-dom";
import { SonnerToastCustom } from './components/custom/soonerToast';
import { useEffect, lazy, Suspense, useRef } from 'react';
import { wsClient } from './api/wsClient';
import { OperationProgressProvider } from './context/OperationProgressContext';
import { PreferencesProvider } from './context/PreferencesContext';
import { AppHealthProvider } from './context/AppHealthContext';
import { UpdateBanner } from './components/custom/UpdateBanner';
import { AuthGate } from './components/auth/AuthGate';
import { VerificationScreen } from './components/auth/VerificationScreen';
import { getSetupStatus } from '@/api/api-auth';

const HomePage = lazy(() => import('./pages/HomePage'));
const ShareVerifyPage = lazy(() => import('./pages/ShareVerifyPage'));
const ShareHomePage = lazy(() => import('./pages/ShareHomePage'));
const IndexPage = lazy(() => import('./pages/IndexPage'));
const NotFoundPage = lazy(() => import('./pages/PageNotFound'));
const LoginPage = lazy(() => import('./pages/LoginPage'));
const AdminPage = lazy(() => import('./pages/AdminPage'));
const AudioBookPage = lazy(() => import('./pages/AudioBookPage'));
const ManageSharesPage = lazy(() => import('./pages/ManageSharesPage'));
const SetupPage = lazy(() => import('./pages/SetupPage'));

function AppLoadingFallback() {
  return <VerificationScreen subtitle="Loading..." />;
}

function App() {
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  navigateRef.current = navigate;

  useEffect(() => {
    const handleAuthUnauthorized = () => {
      if (window.location.pathname !== '/login' && window.location.pathname !== '/setup') {
        void navigateRef.current('/login', { replace: true });
      }
    };
    window.addEventListener('auth:unauthorized', handleAuthUnauthorized);
    return () => { window.removeEventListener('auth:unauthorized', handleAuthUnauthorized); };
  }, []);

  useEffect(() => {
    if (window.location.pathname === '/setup') return;

    getSetupStatus()
      .then((res) => {
        if (res.setup_required) {
          void navigateRef.current('/setup', { replace: true });
        }
      })
      .catch(() => {
        // Setup check failed (e.g. backend not ready), continue normally
      });
  }, []);

  useEffect(() => {
    wsClient.connect();
  }, []);

  return (
    <ThemeProvider defaultTheme="system" storageKey="vite-ui-theme">
      <AppHealthProvider>
        <PreferencesProvider>
          <OperationProgressProvider>
            <UpdateBanner />
            <Suspense fallback={<AppLoadingFallback />}>
              <Routes>
                <Route element={<IndexPage />} path="/" />
                <Route element={<AuthGate />}>
                  <Route path="/home">
                    <Route index element={<HomePage />} />
                    <Route path="*" element={<HomePage />} />
                  </Route>
                  <Route path="/manage-shares" element={<ManageSharesPage />} />
                </Route>
                
                <Route element={<ShareVerifyPage />} path="/share/:id" />
                <Route element={<ShareHomePage />} path="/share/:id/home" />
                <Route element={<ShareHomePage />} path="/share/:id/home/*" />

                <Route element={<LoginPage />} path="/login" />
                <Route element={<SetupPage />} path="/setup" />
                <Route element={<AdminPage />} path="/admin" />
                <Route element={<AudioBookPage />} path="/audio-book" />



                <Route element={<NotFoundPage />} path="*" />
              </Routes>
            </Suspense>
            <SonnerToastCustom />
          </OperationProgressProvider>
        </PreferencesProvider>
      </AppHealthProvider>
    </ThemeProvider>
  )
}

export default App
