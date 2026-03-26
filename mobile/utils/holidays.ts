export interface PublicHoliday {
  date: string;
  localName: string;
  name: string;
  countryCode: string;
}

const cache: Record<string, PublicHoliday[]> = {};

export async function getPublicHolidays(
  year: number,
  countryCode: string
): Promise<PublicHoliday[]> {
  const key = `${year}-${countryCode}`;
  if (cache[key]) return cache[key];
  try {
    const res = await fetch(
      `https://date.nager.at/api/v3/PublicHolidays/${year}/${countryCode}`
    );
    if (!res.ok) return [];
    const holidays: PublicHoliday[] = await res.json();
    cache[key] = holidays;
    return holidays;
  } catch {
    return [];
  }
}

export async function isPublicHoliday(
  date: Date,
  countryCode: string
): Promise<boolean> {
  const holidays = await getPublicHolidays(date.getFullYear(), countryCode);
  const dateStr = date.toISOString().split("T")[0];
  return holidays.some((h) => h.date === dateStr);
}

export async function getUpcomingHolidays(
  countryCode: string,
  withinDays = 14
): Promise<PublicHoliday[]> {
  const now = new Date();
  const holidays = await getPublicHolidays(now.getFullYear(), countryCode);
  const cutoff = new Date(now.getTime() + withinDays * 86400000);
  return holidays.filter((h) => {
    const d = new Date(h.date);
    return d >= now && d <= cutoff;
  });
}

export const COUNTRY_CODE_MAP: Record<string, string> = {
  UAE: "AE",
  "Saudi Arabia": "SA",
  KSA: "SA",
  Kuwait: "KW",
  Qatar: "QA",
  Bahrain: "BH",
  Oman: "OM",
  Egypt: "EG",
  Jordan: "JO",
  Lebanon: "LB",
};

export function inferCountryCode(location: string): string {
  for (const [key, code] of Object.entries(COUNTRY_CODE_MAP)) {
    if (location.toLowerCase().includes(key.toLowerCase())) return code;
  }
  return "AE";
}
