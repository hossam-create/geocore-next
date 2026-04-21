import axios from "axios";

export function getErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as
      | { message?: unknown; error?: unknown; details?: unknown }
      | undefined;

    const fromMessage = typeof data?.message === "string" ? data.message : undefined;
    const fromError = typeof data?.error === "string" ? data.error : undefined;
    const fromDetails = typeof data?.details === "string" ? data.details : undefined;

    return fromMessage || fromError || fromDetails || error.message || fallback;
  }

  const direct = (error as { message?: unknown } | null)?.message;
  return typeof direct === "string" && direct.trim().length > 0 ? direct : fallback;
}
