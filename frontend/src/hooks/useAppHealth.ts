import { useState, useEffect } from "react";
import { checkHealth, type HealthResponse } from "@/api/api-file";
import { wsClient } from "@/api/wsClient";

export function useAppHealth(handleRefresh: () => Promise<void> | void) {
  const [healthData, setHealthData] = useState<HealthResponse | null>(null);
  const [isWsConnected, setIsWsConnected] = useState(false);
  const [isHealthConnected, setIsHealthConnected] = useState(false);

  useEffect(() => {
    const performHealthCheck = async () => {
      const isHealthy = await checkHealth();
      setHealthData(isHealthy);
      setIsHealthConnected(isHealthy !== null);
    };

    const unsubscribeWs = wsClient.subscribeStatus((connected) => {
      setIsWsConnected(connected);
      void performHealthCheck();
    });

    const handleWsReconnect = () => {
      void handleRefresh();
    };
    window.addEventListener('ws-reconnected', handleWsReconnect);

    const healthInterval = setInterval(() => {
      void performHealthCheck();
    }, 30000);

    return () => {
      unsubscribeWs();
      window.removeEventListener('ws-reconnected', handleWsReconnect);
      clearInterval(healthInterval);
    };
  }, [handleRefresh]);

  return { healthData, isWsConnected, isHealthConnected };
}
