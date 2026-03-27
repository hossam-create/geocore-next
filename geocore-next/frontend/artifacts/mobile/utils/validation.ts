export function validatePhoneE164(phone: string): boolean {
  if (!phone) return true;
  return /^\+[1-9]\d{7,14}$/.test(phone);
}

export function validateEmailFormat(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export interface EmailValidationResult {
  valid: boolean;
  disposable: boolean;
  domain: string;
  error?: string;
}

export async function validateEmailDeep(
  email: string
): Promise<EmailValidationResult> {
  if (!validateEmailFormat(email)) {
    return { valid: false, disposable: false, domain: "", error: "Invalid format" };
  }
  try {
    const res = await fetch(
      `https://eva.pingutil.com/email?email=${encodeURIComponent(email)}`,
      { signal: AbortSignal.timeout(4000) }
    );
    if (!res.ok) {
      return { valid: true, disposable: false, domain: "" };
    }
    const body: {
      data: { valid: boolean; disposable: boolean; domain: string };
    } = await res.json();
    return {
      valid: body.data.valid,
      disposable: body.data.disposable,
      domain: body.data.domain,
    };
  } catch {
    return { valid: true, disposable: false, domain: "" };
  }
}

export function getPhoneError(phone: string): string | null {
  if (!phone) return null;
  if (!validatePhoneE164(phone)) {
    return "Phone must be in E.164 format, e.g. +971501234567";
  }
  return null;
}
