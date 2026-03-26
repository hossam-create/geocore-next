import { useState, useCallback } from "react";

export interface LocationData {
  city: string;
  lat: number;
  lon: number;
  countryCode: string;
  country: string;
}

const CACHE_KEY = "geocore_detected_location";
const API_BASE = import.meta.env.BASE_URL.replace(/\/web\/?$/, "/api");

function getCached(): LocationData | null {
  try {
    const cached = sessionStorage.getItem(CACHE_KEY);
    return cached ? (JSON.parse(cached) as LocationData) : null;
  } catch {
    return null;
  }
}

export function useLocation() {
  const [location, setLocation] = useState<LocationData | null>(getCached);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const detectLocation = useCallback(async (forceRefresh = false): Promise<LocationData | null> => {
    if (!forceRefresh) {
      const cached = getCached();
      if (cached) {
        setLocation(cached);
        return cached;
      }
    }

    setLoading(true);
    setError(null);
    try {
      const url = `${API_BASE}/detect-location`;
      const res = await fetch(url);
      if (!res.ok) throw new Error("Location detection failed");
      const json = await res.json() as { data: LocationData };
      const data = json.data;
      setLocation(data);
      sessionStorage.setItem(CACHE_KEY, JSON.stringify(data));
      return data;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to detect location";
      setError(message);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const clearLocation = useCallback(() => {
    setLocation(null);
    sessionStorage.removeItem(CACHE_KEY);
  }, []);

  return { location, loading, error, detectLocation, clearLocation };
}
