import type { AuthRepository } from "../../repositories/auth.repository";
import { SECURE_STORAGE_KEYS } from "../../../core/constants/storageKeys";
import { secureStorage } from "../../../core/services/secure-storage.service";

export class LogoutUseCase {
  constructor(private readonly auth: AuthRepository) {}

  async execute(): Promise<void> {
    await this.auth.logout();
    await secureStorage.remove(SECURE_STORAGE_KEYS.accessToken);
    await secureStorage.remove(SECURE_STORAGE_KEYS.refreshToken);
  }
}
