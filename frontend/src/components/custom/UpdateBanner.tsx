import { useRegisterSW } from 'virtual:pwa-register/react';
import { RefreshCcw, X } from 'lucide-react';
import { Button } from '@/components/ui/button';

export function UpdateBanner() {
  const {
    needRefresh: [needRefresh, setNeedRefresh],
    updateServiceWorker,
  } = useRegisterSW({
    onRegistered(r) {
      console.log('SW Registered:', r);
    },
    onRegisterError(error) {
      console.log('SW registration error', error);
    },
  });

  if (!needRefresh) return null;

  return (
    <div className="fixed top-0 left-0 right-0 z-50 flex items-center justify-between p-3 bg-primary text-white shadow-md animate-in slide-in-from-top">
      <div className="flex items-center gap-2 text-sm font-medium">
        <RefreshCcw className="w-5 h-5 animate-spin-slow" />
        <span>A new version is available! Refresh to update.</span>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          className="h-8"
          onClick={() => { void updateServiceWorker(true); }}
        >
          Refresh
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-white hover:text-white hover:bg-white/20"
          onClick={() => { setNeedRefresh(false); }}
        >
          <X className="w-4 h-4" />
          <span className="sr-only">Dismiss</span>
        </Button>
      </div>
    </div>
  );
}
