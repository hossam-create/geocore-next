/**
 * Backend response envelope — mirrors `pkg/response.R` in the Go API:
 *   { "success": bool, "data": any, "error": string, "meta": any }
 */
export interface ApiEnvelope<T> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: ApiMeta;
}

export interface ApiMeta {
  page?: number;
  per_page?: number;
  total?: number;
  has_more?: boolean;
}

export type ApiResponse<T> = { data: T; meta?: ApiMeta };
