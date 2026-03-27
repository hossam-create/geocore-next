export interface IPInfo {
  country: string;
  countryCode: string;
  city: string;
  lat: number;
  lon: number;
  timezone: string;
}

export async function detectLocationByIP(): Promise<IPInfo | null> {
  try {
    const res = await fetch(
      "https://ip-api.com/json/?fields=country,countryCode,city,lat,lon,timezone"
    );
    if (!res.ok) return null;
    const data: IPInfo & { status: string } = await res.json();
    if (data.status === "fail") return null;
    return data;
  } catch {
    return null;
  }
}

export interface GeoResult {
  lat: string;
  lon: string;
  displayName: string;
}

export async function geocodeAddress(
  query: string
): Promise<GeoResult | null> {
  try {
    const encoded = encodeURIComponent(query);
    const res = await fetch(
      `https://nominatim.openstreetmap.org/search?q=${encoded}&format=json&limit=1&addressdetails=1`,
      {
        headers: {
          "User-Agent": "GeoCore-App/1.0 (geocore@example.com)",
          "Accept-Language": "en",
        },
      }
    );
    if (!res.ok) return null;
    const results: Array<{ lat: string; lon: string; display_name: string }> =
      await res.json();
    if (!results.length) return null;
    return {
      lat: results[0].lat,
      lon: results[0].lon,
      displayName: results[0].display_name,
    };
  } catch {
    return null;
  }
}

export async function reverseGeocode(
  lat: number,
  lon: number
): Promise<string | null> {
  try {
    const res = await fetch(
      `https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lon}&format=json`,
      {
        headers: {
          "User-Agent": "GeoCore-App/1.0 (geocore@example.com)",
          "Accept-Language": "en",
        },
      }
    );
    if (!res.ok) return null;
    const data: {
      address: {
        city?: string;
        town?: string;
        village?: string;
        country?: string;
      };
    } = await res.json();
    const city =
      data.address.city ?? data.address.town ?? data.address.village;
    const country = data.address.country;
    if (city && country) return `${city}, ${country}`;
    if (country) return country;
    return null;
  } catch {
    return null;
  }
}
