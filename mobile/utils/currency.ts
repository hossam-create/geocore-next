const BASE_URL = "https://api.frankfurter.app";

export interface CurrencyMap {
  [code: string]: string;
}

let currencyCache: CurrencyMap | null = null;

export async function fetchSupportedCurrencies(): Promise<CurrencyMap> {
  if (currencyCache) return currencyCache;
  const res = await fetch(`${BASE_URL}/currencies`);
  if (!res.ok) throw new Error("Failed to fetch currencies");
  const data: CurrencyMap = await res.json();
  currencyCache = data;
  return data;
}

export async function convertCurrency(
  amount: number,
  from: string,
  to: string
): Promise<number> {
  if (from === to || amount === 0) return amount;
  const res = await fetch(
    `${BASE_URL}/latest?amount=${amount.toFixed(2)}&from=${from}&to=${to}`
  );
  if (!res.ok) throw new Error(`Currency convert failed: ${res.status}`);
  const data: { rates: Record<string, number> } = await res.json();
  const converted = data.rates[to];
  if (converted === undefined) throw new Error(`No rate for ${to}`);
  return converted;
}

export const GCC_CURRENCIES = [
  { code: "AED", name: "UAE Dirham", symbol: "د.إ" },
  { code: "SAR", name: "Saudi Riyal", symbol: "﷼" },
  { code: "KWD", name: "Kuwaiti Dinar", symbol: "د.ك" },
  { code: "QAR", name: "Qatari Riyal", symbol: "﷼" },
  { code: "BHD", name: "Bahraini Dinar", symbol: ".د.ب" },
  { code: "OMR", name: "Omani Rial", symbol: "﷼" },
  { code: "EGP", name: "Egyptian Pound", symbol: "£" },
  { code: "USD", name: "US Dollar", symbol: "$" },
  { code: "EUR", name: "Euro", symbol: "€" },
  { code: "GBP", name: "British Pound", symbol: "£" },
];

export function getCurrencySymbol(code: string): string {
  return GCC_CURRENCIES.find((c) => c.code === code)?.symbol ?? code;
}

export function formatPrice(amount: number, currency = "AED"): string {
  const symbol = getCurrencySymbol(currency);
  if (amount >= 1_000_000)
    return `${symbol}${(amount / 1_000_000).toFixed(1)}M`;
  if (amount >= 1_000) return `${symbol}${(amount / 1_000).toFixed(1)}K`;
  return `${symbol}${amount.toLocaleString("en-US")}`;
}
