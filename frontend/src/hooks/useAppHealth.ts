import { useAppHealthContext } from "@/context/AppHealthContext";

export function useAppHealth() {
  const { healthData, isWsConnected, isHealthConnected } = useAppHealthContext();
  return { healthData, isWsConnected, isHealthConnected };
}
