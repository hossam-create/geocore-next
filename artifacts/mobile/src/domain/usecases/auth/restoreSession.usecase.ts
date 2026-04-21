import type { AuthRepository } from "../../repositories/auth.repository";
import type { User } from "../../entities";
import { SECURE_STORAGE_KEYS } from "../../../core/constants/storageKeys";
import { secureStorage } from "../../../core/services/secure-storage.service";

export class RestoreSessionUseCase {
  constructor(private readonly auth: AuthRepository) {}

  /**
   * Returns the current user if a valid token exists, otherwise null.
   * Silently clears stale tokens on failure.
   */
  async execute(): Promise<User | null> {
    const token = await secureStorage.get(SECURE_STORAGE_KEYS.accessToken);
    if (!token) return null;
    try {
      return await this.auth.me();
    } catch {
      await secureStorage.remove(SECURE_STORAGE_KEYS.accessToken);
      await secureStorage.remove(SECURE_STORAGE_KEYS.refreshToken);
      return null;
    }
  }
}
