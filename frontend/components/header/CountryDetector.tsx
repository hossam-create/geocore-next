"use client";

import { useEffect, useRef, useState } from "react";
import { ChevronDown, X, Globe } from "lucide-react";
import { useCountry, useCountryList, getStoredCountryCode, setStoredCountryCode } from "@/lib/useCountry";

const GCC_COUNTRIES = [
  { code: "EG", name: "Egypt", nameAr: "مصر", currency: "EGP", symbol: "E£" },
  { code: "SA", name: "Saudi Arabia", nameAr: "السعودية", currency: "SAR", symbol: "﷼" },
  { code: "AE", name: "UAE", nameAr: "الإمارات", currency: "AED", symbol: "د.إ" },
  { code: "KW", name: "Kuwait", nameAr: "الكويت", currency: "KWD", symbol: "د.ك" },
  { code: "BH", name: "Bahrain", nameAr: "البحرين", currency: "BHD", symbol: "د.ب" },
  { code: "QA", name: "Qatar", nameAr: "قطر", currency: "QAR", symbol: "ر.ق" },
  { code: "OM", name: "Oman", nameAr: "عمان", currency: "OMR", symbol: "ر.ع" },
];

export function CountryDetector() {
  const [open, setOpen] = useState(false);
  const [showAlert, setShowAlert] = useState(false);
  const [detectedCode, setDetectedCode] = useState("");
  const ref = useRef<HTMLDivElement>(null);

  const storedCode = getStoredCountryCode();
  const { config } = useCountry(storedCode);
  const { countries: apiCountries } = useCountryList();

  const currentCode = config?.country_code || storedCode;
  const currentCountry = GCC_COUNTRIES.find(c => c.code === currentCode) || GCC_COUNTRIES[0];
  const flagUrl = `https://flagcdn.com/20x15/${currentCode.toLowerCase()}.png`;

  // Detect country from IP on first load
  useEffect(() => {
    const cached = sessionStorage.getItem("detected_country");
    if (cached) {
      try {
        const parsed = JSON.parse(cached);
        if (parsed?.code) setDetectedCode(parsed.code);
        return;
      } catch {}
    }
    fetch("https://ipapi.co/json/")
      .then((r) => r.json())
      .then((d) => {
        if (d?.country_code) {
          setDetectedCode(d.country_code);
          sessionStorage.setItem("detected_country", JSON.stringify({ code: d.country_code, name: d.country_name }));
          // If no stored preference, auto-set
          if (!localStorage.getItem("mnbarh_country_code")) {
            const match = GCC_COUNTRIES.find(c => c.code === d.country_code);
            if (match) {
              setStoredCountryCode(d.country_code);
            }
          }
          if (d.country_code !== storedCode) setShowAlert(true);
        }
      })
      .catch(() => {});
  }, [storedCode]);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const switchCountry = (code: string) => {
    setStoredCountryCode(code);
    setOpen(false);
    setShowAlert(false);
    window.location.reload();
  };

  return (
    <>
      {showAlert && detectedCode && detectedCode !== currentCode && (
        <div className="absolute top-full left-0 right-0 z-[100] bg-amber-500 text-white text-xs px-4 py-2 flex items-center justify-between gap-4 shadow-md">
          <span>
            📍 Looks like you&apos;re in <strong>{GCC_COUNTRIES.find(c => c.code === detectedCode)?.name || detectedCode}</strong> — your store is set to {currentCountry.name}.
            <button onClick={() => switchCountry(detectedCode)} className="ml-2 underline font-semibold hover:text-amber-100">Switch to {detectedCode}</button>
          </span>
          <button onClick={() => setShowAlert(false)} className="shrink-0 hover:opacity-70">
            <X size={14} />
          </button>
        </div>
      )}

      <div className="relative" ref={ref}>
        <button
          onClick={() => setOpen((v) => !v)}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border border-white/25 bg-white/10 text-white hover:bg-white/15 transition-colors"
          title={`${currentCountry.name} · ${config?.currency_symbol || currentCountry.symbol}`}
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={flagUrl}
            alt={currentCode}
            width={18}
            height={13}
            className="rounded-sm shadow-sm"
            onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
          />
          <span className="text-[10px] font-semibold">{config?.currency_symbol || currentCountry.symbol}</span>
          <ChevronDown size={10} className={`transition-transform duration-200 ${open ? "rotate-180" : ""}`} />
        </button>

        {open && (
          <div className="absolute right-0 top-full mt-2 w-72 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800 overflow-hidden">
            <div className="px-4 py-3 border-b border-gray-100 flex items-center gap-2">
              <Globe size={14} className="text-[#0071CE]" />
              <div>
                <p className="text-sm font-bold text-gray-900">Select Country</p>
                <p className="text-[11px] text-gray-500">Prices shown in local currency</p>
              </div>
            </div>
            <div className="py-2 max-h-72 overflow-y-auto">
              {(apiCountries.length > 0 ? apiCountries : GCC_COUNTRIES).map((raw) => {
                const c = raw as Record<string, string | boolean | undefined>;
                const code = String(c.code);
                const name = String(c.name_en || c.name || code);
                const nameAr = c.name_ar ? String(c.name_ar) : c.nameAr ? String(c.nameAr) : "";
                const symbol = String(c.currency_symbol || c.symbol || "");
                const isActive = code === currentCode;
                const cFlag = `https://flagcdn.com/24x18/${code.toLowerCase()}.png`;

                return (
                  <button
                    key={code}
                    onClick={() => switchCountry(code)}
                    className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                      isActive ? "bg-blue-50" : "hover:bg-gray-50"
                    }`}
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={cFlag} alt={code} width={24} height={18} className="rounded-sm shadow-sm" />
                    <div className="flex-1">
                      <p className={`text-sm font-medium ${isActive ? "text-[#0071CE]" : "text-gray-800"}`}>{name}</p>
                      {nameAr && <p className="text-[11px] text-gray-400">{nameAr}</p>}
                    </div>
                    <span className="text-xs font-semibold text-gray-500">{symbol}</span>
                    {isActive && <div className="w-2 h-2 rounded-full bg-[#0071CE]" />}
                  </button>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </>
  );
}
