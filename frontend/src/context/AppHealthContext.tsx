import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { checkHealth, type HealthResponse } from "@/api/api-file";
import { wsClient } from "@/api/wsClient";

interface AppHealthContextProps {
  healthData: HealthResponse | null;
  isWsConnected: boolean;
  isHealthConnected: boolean;
  refreshHealth: () => Promise<void>;
}

const AppHealthContext = createContext<AppHealthContextProps | undefined>(undefined);

export function AppHealthProvider({ children }: { children: ReactNode }) {
  const [healthData, setHealthData] = useState<HealthResponse | null>(null);
  const [isWsConnected, setIsWsConnected] = useState(false);
  const [isHealthConnected, setIsHealthConnected] = useState(false);

  const performHealthCheck = async () => {
    const isHealthy = await checkHealth();
    setHealthData(isHealthy);
    setIsHealthConnected(isHealthy !== null);
  };

  useEffect(() => {
    if (healthData?.service_name) {
      document.title = healthData.service_name;
    } else {
      document.title = "GoNet Drive";
    }
  }, [healthData?.service_name]);

  useEffect(() => {
    void performHealthCheck();

    const unsubscribeWs = wsClient.subscribeStatus((connected) => {
      setIsWsConnected(connected);
      void performHealthCheck();
    });

    const healthInterval = setInterval(() => {
      void performHealthCheck();
    }, 30000);

    return () => {
      unsubscribeWs();
      clearInterval(healthInterval);
    };
  }, []);

  return (
    <AppHealthContext.Provider value={{ healthData, isWsConnected, isHealthConnected, refreshHealth: performHealthCheck }}>
      {children}
    </AppHealthContext.Provider>
  );
}

export function useAppHealthContext() {
  const context = useContext(AppHealthContext);
  if (context === undefined) {
    throw new Error('useAppHealthContext must be used within an AppHealthProvider');
  }
  return context;
}
