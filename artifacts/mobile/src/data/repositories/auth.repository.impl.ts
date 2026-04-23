import type {
  AuthRepository,
  LoginCredentials,
  RegisterPayload,
} from "../../domain/repositories/auth.repository";
import type { Session, User } from "../../domain/entities";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { AUTH_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope } from "../api/types";
import { toUser, type ApiUser } from "../mappers/user.mapper";

interface ApiAuthPayload {
  access_token: string;
  refresh_token: string;
  expires_at?: string | null;
  user: ApiUser;
}

function toSession(api: ApiAuthPayload): Session {
  return {
    user: toUser(api.user),
    tokens: {
      accessToken: api.access_token,
      refreshToken: api.refresh_token,
      expiresAt: api.expires_at ?? undefined,
    },
  };
}

export class HttpAuthRepository implements AuthRepository {
  constructor(private readonly http: HttpClient) {}

  async login(credentials: LoginCredentials): Promise<Session> {
    const { data } = await this.http.instance.post<ApiEnvelope<ApiAuthPayload>>(
      AUTH_ENDPOINTS.login,
      credentials,
    );
    return toSession(unwrapEnvelope(data));
  }

  async register(payload: RegisterPayload): Promise<Session> {
    const { data } = await this.http.instance.post<ApiEnvelope<ApiAuthPayload>>(
      AUTH_ENDPOINTS.register,
      payload,
    );
    return toSession(unwrapEnvelope(data));
  }

  async logout(): Promise<void> {
    try {
      await this.http.instance.post(AUTH_ENDPOINTS.logout);
    } catch {
      // logout is best-effort; still clear local state
    }
  }

  async me(): Promise<User> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiUser>>(
      AUTH_ENDPOINTS.me,
    );
    return toUser(unwrapEnvelope(data));
  }

  async refresh(refreshToken: string): Promise<Session> {
    const { data } = await this.http.instance.post<ApiEnvelope<ApiAuthPayload>>(
      AUTH_ENDPOINTS.refresh,
      { refresh_token: refreshToken },
    );
    return toSession(unwrapEnvelope(data));
  }

  async requestPasswordReset(email: string): Promise<void> {
    await this.http.instance.post(AUTH_ENDPOINTS.forgotPassword, { email });
  }

  async verifyOtp(identifier: string, code: string): Promise<void> {
    await this.http.instance.post(AUTH_ENDPOINTS.verifyOtp, {
      identifier,
      code,
    });
  }

  async resendOtp(identifier: string): Promise<void> {
    await this.http.instance.post(AUTH_ENDPOINTS.resendOtp, { identifier });
  }
}
