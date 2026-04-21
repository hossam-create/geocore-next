import { getRequestConfig } from "next-intl/server";
import { cookies } from "next/headers";

const LOCALE_MAP: Record<string, string> = {
  EG: "ar",
  SA: "ar",
  AE: "ar",
  KW: "ar",
  BH: "ar",
  QA: "ar",
  OM: "ar",
};

export default getRequestConfig(async () => {
  let locale = "en";
  try {
    const cookieStore = await cookies();
    const localeCookie = cookieStore.get("NEXT_LOCALE")?.value;
    if (localeCookie && ["en", "ar"].includes(localeCookie)) {
      locale = localeCookie;
    } else {
      const acceptLang = (await cookieStore.get("accept-language"))?.value || "";
      if (acceptLang.startsWith("ar")) {
        locale = "ar";
      }
    }
  } catch {
    // Fallback to English
  }

  return {
    locale,
    messages: (await import(`./messages/${locale}.json`)).default,
  };
});
