import { formatDistanceToNow, parseISO } from "date-fns";

/**
 * Returns a human-readable relative timestamp like "2 min ago".
 * Falls back to ISO date string on parse failure.
 */
export function timeAgo(dateInput: string | Date | undefined | null): string {
  if (!dateInput) return "—";
  try {
    const date = typeof dateInput === "string" ? parseISO(dateInput) : dateInput;
    return formatDistanceToNow(date, { addSuffix: true });
  } catch {
    return typeof dateInput === "string" ? dateInput : "—";
  }
}

/**
 * Mask an email address for PII protection.
 * "john.doe@example.com" → "jo***@example.com"
 */
export function maskEmail(email: string | undefined | null): string {
  if (!email || !email.includes("@")) return "—";
  const [local, domain] = email.split("@");
  const visible = local.length <= 2 ? local.charAt(0) : local.slice(0, 2);
  return `${visible}***@${domain}`;
}

/**
 * Mask a phone number for PII protection.
 * "+1234567890" → "+1***7890"
 */
export function maskPhone(phone: string | undefined | null): string {
  if (!phone || phone.length < 6) return phone ?? "—";
  const last4 = phone.slice(-4);
  const prefix = phone.slice(0, 2);
  return `${prefix}***${last4}`;
}
