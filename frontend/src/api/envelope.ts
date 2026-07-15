import type { AxiosResponse } from "axios";

export interface ApiEnvelope<T> {
  status: "success" | "error";
  data?: T;
  meta?: unknown;
  error?: string;
}

export class ApiError extends Error {
  code: number;
  constructor(code: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

// Common case: return typed data; throw as a safety-net if a 2xx is not "success".
export function unwrap<T>(res: AxiosResponse<ApiEnvelope<T>>): T {
  if (res.data.status !== "success") {
    throw new ApiError(res.status, res.data.error ?? "unknown error");
  }
  return res.data.data ?? ({} as T);
}

// When a caller needs pagination/meta, return both (meta is optional/forward-compat).
export function unwrapWithMeta<T>(
  res: AxiosResponse<ApiEnvelope<T>>,
): { data: T; meta?: unknown } {
  return { data: unwrap<T>(res), meta: res.data.meta };
}
