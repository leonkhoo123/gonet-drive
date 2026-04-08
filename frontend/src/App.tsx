import './App.css'
import { ThemeProvider } from './components/theme-provider'
import { Route, Routes } from "react-router-dom";
import { SonnerToastCustom } from './components/custom/soonerToast';
import { useEffect, lazy, Suspense } from 'react';
import { wsClient } from './api/wsClient';
import { OperationProgressProvider } from './context/OperationProgressContext';
import { PreferencesProvider } from './context/PreferencesContext';
import { AppHealthProvider } from './context/AppHealthContext';
import { UpdateBanner } from './components/custom/UpdateBanner';
import { Loader2 } from "lucide-react";

const HomePage = lazy(() => import('./pages/HomePage'));
const ShareVerifyPage = lazy(() => import('./pages/ShareVerifyPage'));
const ShareHomePage = lazy(() => import('./pages/ShareHomePage'));
const IndexPage = lazy(() => import('./pages/IndexPage'));
const NotFoundPage = lazy(() => import('./pages/PageNotFound'));
const LoginPage = lazy(() => import('./pages/LoginPage'));
const AdminPage = lazy(() => import('./pages/AdminPage'));
const AudioBookPage = lazy(() => import('./pages/AudioBookPage'));
const ManageSharesPage = lazy(() => import('./pages/ManageSharesPage'));

function AppLoadingFallback() {
  return (
    <div className="flex h-screen w-screen items-center justify-center bg-background">
      <Loader2 className="h-8 w-8 animate-spin text-primary" />
    </div>
  );
}

function App() {
  useEffect(() => {
    wsClient.connect();
  }, []);

  return (
    <ThemeProvider defaultTheme="light" storageKey="vite-ui-theme">
      <AppHealthProvider>
        <PreferencesProvider>
          <OperationProgressProvider>
            <UpdateBanner />
            <Suspense fallback={<AppLoadingFallback />}>
              <Routes>
                <Route element={<IndexPage />} path="/" />
                <Route element={<HomePage />} path="/home" />
                <Route element={<HomePage />} path="/home/*" />
                
                <Route element={<ShareVerifyPage />} path="/share/:id" />
                <Route element={<ShareHomePage />} path="/share/:id/home" />
                <Route element={<ShareHomePage />} path="/share/:id/home/*" />

                <Route element={<LoginPage />} path="/login" />
                <Route element={<AdminPage />} path="/admin" />
                <Route element={<AudioBookPage />} path="/audio-book" />
                <Route element={<ManageSharesPage />} path="/manage-shares" />



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
