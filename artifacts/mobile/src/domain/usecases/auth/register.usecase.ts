import type { Session } from "../../entities";
import type {
  AuthRepository,
  RegisterPayload,
} from "../../repositories/auth.repository";
import { regex } from "../../../core/constants/regexPatterns";
import { LIMITS } from "../../../config/constants";
import { ValidationError } from "../../../core/utils/errors";

export class RegisterUseCase {
  constructor(private readonly auth: AuthRepository) {}

  async execute(input: RegisterPayload): Promise<Session> {
    const fields: Record<string, string> = {};
    if (!input.name || input.name.trim().length < 2) {
      fields.name = "Name is required";
    }
    if (!regex.email.test(input.email)) {
      fields.email = "Enter a valid email address";
    }
    if (input.phone && !regex.phone.test(input.phone)) {
      fields.phone = "Enter a valid phone number";
    }
    if (!input.password || input.password.length < LIMITS.minPasswordLength) {
      fields.password = `Password must be at least ${LIMITS.minPasswordLength} characters`;
    }
    if (Object.keys(fields).length > 0) {
      throw new ValidationError("Invalid registration details", fields);
    }
    return this.auth.register(input);
  }
}
