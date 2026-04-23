import type { Session } from "../../entities";
import type {
  AuthRepository,
  LoginCredentials,
} from "../../repositories/auth.repository";
import { regex } from "../../../core/constants/regexPatterns";
import { ValidationError } from "../../../core/utils/errors";

export class LoginUseCase {
  constructor(private readonly auth: AuthRepository) {}

  async execute(input: LoginCredentials): Promise<Session> {
    const fields: Record<string, string> = {};
    if (!input.email || !regex.email.test(input.email)) {
      fields.email = "Enter a valid email address";
    }
    if (!input.password || input.password.length < 6) {
      fields.password = "Password must be at least 6 characters";
    }
    if (Object.keys(fields).length > 0) {
      throw new ValidationError("Invalid login details", fields);
    }
    return this.auth.login(input);
  }
}
