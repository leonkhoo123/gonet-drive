import axios from "@/api/axiosLayer";
import { unwrap, type ApiEnvelope } from "@/api/envelope";

export interface AuditLogEntry {
  id: number;
  event_type: string;
  username: string;
  ip_address: string;
  device_info: string;
  family_id: string;
  metadata: string;
  timestamp: string;
}

export interface AuditLogQuery {
  page?: number;
  page_size?: number;
  event_type?: string;
  username?: string;
}

export interface AuditLogPage {
  logs: AuditLogEntry[];
  total: number;
  page: number;
  page_size: number;
}

export const getAuditLogs = async (query: AuditLogQuery): Promise<AuditLogPage> => {
  const response = await axios.get<ApiEnvelope<AuditLogPage>>("/user/admin/audit-logs", {
    params: query,
  });
  return unwrap<AuditLogPage>(response);
};
