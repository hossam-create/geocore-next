"use client";

import { useTransition } from "react";
import { useLocale } from "next-intl";
import { Globe } from "lucide-react";

export function LanguageSwitcher() {
  const locale = useLocale();
  const [isPending, startTransition] = useTransition();

  const switchLocale = (newLocale: string) => {
    document.cookie = `NEXT_LOCALE=${newLocale};path=/;max-age=31536000`;
    startTransition(() => {
      window.location.reload();
    });
  };

  const isArabic = locale === "ar";

  return (
    <button
      onClick={() => switchLocale(isArabic ? "en" : "ar")}
      disabled={isPending}
      className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border border-white/25 bg-white/10 text-white hover:bg-white/15 transition-colors disabled:opacity-50"
      title={isArabic ? "Switch to English" : "التحول للعربي"}
    >
      <Globe size={12} />
      <span className="text-[10px] font-semibold">{isArabic ? "EN" : "عربي"}</span>
    </button>
  );
}
