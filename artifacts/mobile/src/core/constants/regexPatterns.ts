export const regex = {
  email: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
  // E.164 international phone (8-15 digits, optional leading +)
  phone: /^\+?[1-9]\d{7,14}$/,
  // At least one letter, one digit, min 8 chars.
  strongPassword: /^(?=.*[A-Za-z])(?=.*\d).{8,}$/,
  uuid: /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
} as const;
