import axios from "@/api/axiosLayer";

export interface ConfigItem {
  id: number;
  config_name: string;
  config_type: string;
  config_unit: string | null;
  config_value: string | null;
  is_enabled: boolean;
}

export const getConfigs = async (): Promise<ConfigItem[]> => {
  const response = await axios.get<ConfigItem[]>("/user/config");
  return response.data;
};

export const updateConfig = async (
  id: number,
  data: { config_value?: string | null; is_enabled?: boolean; is_deleted?: boolean }
): Promise<ConfigItem> => {
  const response = await axios.put<ConfigItem>(`/user/config/${id}`, data);
  return response.data;
};
