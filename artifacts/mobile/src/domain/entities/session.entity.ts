import type { User } from "./user.entity";

export interface AuthTokens {
  readonly accessToken: string;
  readonly refreshToken: string;
  readonly expiresAt?: string;
}

export interface Session {
  readonly user: User;
  readonly tokens: AuthTokens;
}
