import type { AxiosError } from "axios";

import {
  AppError,
  ForbiddenError,
  NetworkError,
  NotFoundError,
  UnauthorizedError,
  ValidationError,
} from "../../core/utils/errors";

interface ErrorBody {
  error?: string;
  message?: string;
  fields?: Record<string, string>;
}

export function toAppError(error: unknown): AppError {
  if (error instanceof AppError) return error;

  const axiosErr = error as AxiosError<ErrorBody>;
  if (axiosErr?.isAxiosError) {
    const status = axiosErr.response?.status ?? 0;
    const body = axiosErr.response?.data;
    const msg = body?.error ?? body?.message ?? axiosErr.message;

    switch (status) {
      case 0:
        return new NetworkError(msg, error);
      case 401:
        return new UnauthorizedError(msg);
      case 403:
        return new ForbiddenError(msg);
      case 404:
        return new NotFoundError(msg ?? "Resource");
      case 422:
      case 400:
        return new ValidationError(msg ?? "Invalid request", body?.fields ?? {});
      default:
        return new AppError(`HTTP_${status}`, msg, error);
    }
  }

  return new AppError(
    "UNKNOWN",
    error instanceof Error ? error.message : String(error),
    error,
  );
}
