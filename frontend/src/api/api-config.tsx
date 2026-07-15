import axios from "@/api/axiosLayer";
import { unwrap, type ApiEnvelope } from "@/api/envelope";

export interface ConfigItem {
  id: number;
  config_name: string;
  config_type: string;
  config_unit: string | null;
  config_value: string | null;
  is_enabled: boolean;
}

export const getConfigs = async (): Promise<ConfigItem[]> => {
  const response = await axios.get<ApiEnvelope<{ configs: ConfigItem[] }>>("/user/config");
  return unwrap<{ configs: ConfigItem[] }>(response).configs;
};

export const updateConfig = async (
  id: number,
  data: { config_value?: string | null; is_enabled?: boolean; is_deleted?: boolean }
): Promise<void> => {
  const response = await axios.put<ApiEnvelope<unknown>>(`/user/config/${String(id)}`, data);
  unwrap<unknown>(response);
};
