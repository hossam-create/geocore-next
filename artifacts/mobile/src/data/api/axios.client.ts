import axios, {
  type AxiosInstance,
  type InternalAxiosRequestConfig,
} from "axios";

import { apiConfig } from "../../config/api.config";
import { SECURE_STORAGE_KEYS } from "../../core/constants/storageKeys";
import { secureStorage } from "../../core/services/secure-storage.service";
import { toAppError } from "./http-error";
import { AUTH_ENDPOINTS } from "./endpoints";
import type { ApiEnvelope } from "./types";

type RetriableConfig = InternalAxiosRequestConfig & { _retry?: boolean };

export interface HttpClient {
  readonly instance: AxiosInstance;
  onUnauthorized(handler: () => void | Promise<void>): () => void;
}

function unwrap<T>(body: ApiEnvelope<T> | T): T {
  if (body && typeof body === "object" && "success" in body) {
    const env = body as ApiEnvelope<T>;
    if (env.success === false) {
      throw new Error(env.error ?? "Request failed");
    }
    return (env.data ?? (undefined as unknown)) as T;
  }
  return body as T;
}

export function unwrapEnvelope<T>(body: ApiEnvelope<T> | T): T {
  return unwrap(body);
}

export function createHttpClient(): HttpClient {
  const instance = axios.create(apiConfig);
  const unauthorizedHandlers = new Set<() => void | Promise<void>>();

  instance.interceptors.request.use(async (config) => {
    const token = await secureStorage.get(SECURE_STORAGE_KEYS.accessToken);
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });

  instance.interceptors.response.use(
    (res) => res,
    async (error) => {
      const original = error.config as RetriableConfig | undefined;

      if (error.response?.status === 401 && original && !original._retry) {
        original._retry = true;
        const refresh = await secureStorage.get(
          SECURE_STORAGE_KEYS.refreshToken,
        );

        if (refresh) {
          try {
            const { data } = await axios.post(
              `${apiConfig.baseURL}${AUTH_ENDPOINTS.refresh}`,
              { refresh_token: refresh },
              { headers: apiConfig.headers, timeout: apiConfig.timeout },
            );
            const payload = unwrap<{
              access_token: string;
              refresh_token?: string;
            }>(data);

            await secureStorage.set(
              SECURE_STORAGE_KEYS.accessToken,
              payload.access_token,
            );
            if (payload.refresh_token) {
              await secureStorage.set(
                SECURE_STORAGE_KEYS.refreshToken,
                payload.refresh_token,
              );
            }

            original.headers.Authorization = `Bearer ${payload.access_token}`;
            return instance(original);
          } catch (refreshErr) {
            await secureStorage.remove(SECURE_STORAGE_KEYS.accessToken);
            await secureStorage.remove(SECURE_STORAGE_KEYS.refreshToken);
            for (const handler of unauthorizedHandlers) {
              await handler();
            }
            return Promise.reject(toAppError(refreshErr));
          }
        }

        // No refresh token — notify listeners and bubble up.
        for (const handler of unauthorizedHandlers) {
          await handler();
        }
      }

      return Promise.reject(toAppError(error));
    },
  );

  return {
    instance,
    onUnauthorized(handler) {
      unauthorizedHandlers.add(handler);
      return () => unauthorizedHandlers.delete(handler);
    },
  };
}

export const http = createHttpClient();
