import { useEffect } from "react";
import { useAppHealthContext } from "@/context/AppHealthContext";

export function useAppHealth(handleRefresh?: () => Promise<void> | void) {
  const { healthData, isWsConnected, isHealthConnected } = useAppHealthContext();

  useEffect(() => {
    if (!handleRefresh) return;
    
    const handleWsReconnect = () => {
      void handleRefresh();
    };
    window.addEventListener('ws-reconnected', handleWsReconnect);

    return () => {
      window.removeEventListener('ws-reconnected', handleWsReconnect);
    };
  }, [handleRefresh]);

  return { healthData, isWsConnected, isHealthConnected };
}
