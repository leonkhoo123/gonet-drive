export interface AppConfig {
  apiBaseUrl: string;
  wsUrl: string;
}

// Default configuration with the logic currently used in the app
const isLocal = import.meta.env.VITE_BUILD_PROFILE === "local";
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

const config: AppConfig = {
  apiBaseUrl: isLocal ? "http://localhost:3333/api" : "/api",
  wsUrl: isLocal ? "ws://localhost:3333/api/user/ws" : `${protocol}//${window.location.host}/api/user/ws`,
};

export const loadConfig = async (): Promise<void> => {
  try {
    const response = await fetch('/config.json');
    if (response.ok) {
      // Type assertion to let TypeScript know what we expect
      const data = (await response.json()) as Partial<AppConfig>;
      
      // Only override if the value is explicitly provided in the JSON
      if (data.apiBaseUrl) {
        config.apiBaseUrl = data.apiBaseUrl;
      }
      if (data.wsUrl) {
        config.wsUrl = data.wsUrl;
      }
      console.log("[Config] Loaded configuration from /config.json");
    } else {
      console.warn("[Config] Could not load /config.json, using fallback defaults.");
    }
  } catch (error) {
    console.warn("[Config] Failed to fetch /config.json, using fallback defaults.", error);
  }
};

export const getConfig = (): AppConfig => config;
