export class AppError extends Error {
  public readonly code: string;
  public readonly cause?: unknown;

  constructor(code: string, message: string, cause?: unknown) {
    super(message);
    this.name = "AppError";
    this.code = code;
    this.cause = cause;
  }
}

export class NetworkError extends AppError {
  constructor(message = "Network request failed", cause?: unknown) {
    super("NETWORK_ERROR", message, cause);
    this.name = "NetworkError";
  }
}

export class UnauthorizedError extends AppError {
  constructor(message = "Not authenticated") {
    super("UNAUTHORIZED", message);
    this.name = "UnauthorizedError";
  }
}

export class ForbiddenError extends AppError {
  constructor(message = "Not allowed") {
    super("FORBIDDEN", message);
    this.name = "ForbiddenError";
  }
}

export class NotFoundError extends AppError {
  constructor(entity: string) {
    super("NOT_FOUND", `${entity} not found`);
    this.name = "NotFoundError";
  }
}

export class ValidationError extends AppError {
  public readonly fields: Readonly<Record<string, string>>;

  constructor(
    message: string,
    fields: Record<string, string> = {},
  ) {
    super("VALIDATION_ERROR", message);
    this.name = "ValidationError";
    this.fields = fields;
  }
}
