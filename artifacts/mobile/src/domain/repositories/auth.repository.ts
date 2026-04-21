import type { Session, User } from "../entities";

export interface LoginCredentials {
  readonly email: string;
  readonly password: string;
}

export interface RegisterPayload {
  readonly name: string;
  readonly email: string;
  readonly password: string;
  readonly phone?: string;
}

export interface AuthRepository {
  login(credentials: LoginCredentials): Promise<Session>;
  register(payload: RegisterPayload): Promise<Session>;
  logout(): Promise<void>;
  me(): Promise<User>;
  refresh(refreshToken: string): Promise<Session>;
  requestPasswordReset(email: string): Promise<void>;
  verifyOtp(identifier: string, code: string): Promise<void>;
  resendOtp(identifier: string): Promise<void>;
}
