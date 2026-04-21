"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

export interface ResolvedCountryConfig {
  country_code: string;
  currency: string;
  currency_ar?: string;
  currency_symbol?: string;
  tax_rate: number;
  tax_label: string;
  tax_inclusive: boolean;
  service_fee_rate: number;
  withholding_rate: number;
  kyc_tier1_limit_cents: number;
  kyc_tier2_limit_cents: number;
  kyc_tier3_limit_cents: number;
  max_listing_price_cents: number;
  payment_methods: string[];
  enable_auctions: boolean;
  enable_live: boolean;
  enable_bnpl: boolean;
  enable_p2p: boolean;
  enable_crypto: boolean;
  enable_crowdship: boolean;
  enable_wholesale: boolean;
  enable_real_estate: boolean;
  default_shipping_cents: number;
  free_shipping_threshold_cents: number;
  locale: string;
  require_national_id: boolean;
  require_address_proof: boolean;
  min_age: number;
  max_return_days: number;
}

const COUNTRY_KEY = "mnbarh_country_code";

export function getStoredCountryCode(): string {
  if (typeof window === "undefined") return "EG";
  return localStorage.getItem(COUNTRY_KEY) || "EG";
}

export function setStoredCountryCode(code: string) {
  if (typeof window === "undefined") return;
  localStorage.setItem(COUNTRY_KEY, code);
}

export function useCountry(code?: string) {
  const countryCode = code || getStoredCountryCode();

  const { data, isLoading, error } = useQuery({
    queryKey: ["country", countryCode],
    queryFn: async () => {
      try {
        const res = await fetch(`/api/v1/country/${countryCode}`);
        if (!res.ok) throw new Error("not found");
        const json = await res.json();
        return json.data as ResolvedCountryConfig;
      } catch {
        // Fallback defaults
        return {
          country_code: countryCode,
          currency: "EGP",
          currency_ar: "جنيه",
          currency_symbol: "E£",
          tax_rate: 0.14,
          tax_label: "VAT",
          tax_inclusive: true,
          service_fee_rate: 0.05,
          withholding_rate: 0,
          kyc_tier1_limit_cents: 500000,
          kyc_tier2_limit_cents: 5000000,
          kyc_tier3_limit_cents: 0,
          max_listing_price_cents: 0,
          payment_methods: ["card", "cash_on_delivery", "wallet", "paymob", "bnpl"],
          enable_auctions: true,
          enable_live: true,
          enable_bnpl: true,
          enable_p2p: false,
          enable_crypto: false,
          enable_crowdship: true,
          enable_wholesale: true,
          enable_real_estate: true,
          default_shipping_cents: 5000,
          free_shipping_threshold_cents: 100000,
          locale: "ar-EG",
          require_national_id: true,
          require_address_proof: false,
          min_age: 18,
          max_return_days: 14,
        } as ResolvedCountryConfig;
      }
    },
    staleTime: 5 * 60 * 1000, // 5 min cache
  });

  return { config: data, isLoading, error, countryCode };
}

export function useCountryList() {
  const { data, isLoading } = useQuery({
    queryKey: ["countries"],
    queryFn: async () => {
      try {
        const res = await fetch("/api/v1/country");
        if (!res.ok) throw new Error("failed");
        const json = await res.json();
        return json.data as Array<{
          code: string;
          name_en: string;
          name_ar?: string;
          currency: string;
          currency_symbol?: string;
          is_active: boolean;
        }>;
      } catch {
        return [
          { code: "EG", name_en: "Egypt", name_ar: "مصر", currency: "EGP", currency_symbol: "E£", is_active: true },
          { code: "SA", name_en: "Saudi Arabia", name_ar: "المملكة العربية السعودية", currency: "SAR", currency_symbol: "﷼", is_active: true },
          { code: "AE", name_en: "UAE", name_ar: "الإمارات", currency: "AED", currency_symbol: "د.إ", is_active: true },
          { code: "KW", name_en: "Kuwait", name_ar: "الكويت", currency: "KWD", currency_symbol: "د.ك", is_active: true },
          { code: "BH", name_en: "Bahrain", name_ar: "البحرين", currency: "BHD", currency_symbol: "د.ب", is_active: true },
          { code: "QA", name_en: "Qatar", name_ar: "قطر", currency: "QAR", currency_symbol: "ر.ق", is_active: true },
          { code: "OM", name_en: "Oman", name_ar: "عمان", currency: "OMR", currency_symbol: "ر.ع", is_active: true },
        ];
      }
    },
    staleTime: 10 * 60 * 1000,
  });

  return { countries: data ?? [], isLoading };
}

/** Format price using the country's currency symbol */
export function formatCountryPrice(amountCents: number, config?: ResolvedCountryConfig): string {
  if (!config) return `${amountCents / 100}`;
  const amount = amountCents / 100;
  const symbol = config.currency_symbol || config.currency;
  try {
    return new Intl.NumberFormat(config.locale || "en-US", {
      style: "currency",
      currency: config.currency,
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  } catch {
    return `${symbol}${amount.toLocaleString()}`;
  }
}
