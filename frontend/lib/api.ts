import axios from "axios";

const BASE_URL = "/api/v1";

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  headers: { "Content-Type": "application/json" },
});

export interface CartItem {
  listing_id: string;
  title: string;
  image_url?: string;
  currency: string;
  unit_price: number;
  quantity: number;
  subtotal: number;
}

export interface CartData {
  items: CartItem[];
  item_count: number;
  total: number;
  currency?: string;
}

export interface WatchlistItem {
  id: string;
  title: string;
  price?: number;
  currency?: string;
  image_url?: string;
  images?: Array<{ id?: string; url: string }>;
  city?: string;
  type?: string;
  is_auction?: boolean;
  is_watched?: boolean;
  created_at?: string;
}

const WATCHLIST_PRICE_KEY = "watchlist_price_snapshot";

function getWatchlistPriceMap(): Record<string, number> {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(WATCHLIST_PRICE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, number>;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function setWatchlistPriceMap(map: Record<string, number>) {
  if (typeof window === "undefined") return;
  localStorage.setItem(WATCHLIST_PRICE_KEY, JSON.stringify(map));
}

export async function getCart() {
  const res = await api.get("/cart");
  return res.data?.data as CartData;
}

export async function addCartItem(listingID: string, quantity = 1) {
  const res = await api.post("/cart/items", { listing_id: listingID, quantity });
  return res.data?.data as CartData;
}

export async function removeCartItem(listingID: string) {
  const res = await api.delete(`/cart/items/${listingID}`);
  return res.data?.data;
}

export async function clearCart() {
  const res = await api.delete("/cart");
  return res.data?.data;
}

export async function getWatchlist(page = 1, perPage = 20) {
  const res = await api.get(`/watchlist?page=${page}&per_page=${perPage}`);
  return {
    items: (res.data?.data ?? []) as WatchlistItem[],
    meta: res.data?.meta,
  };
}

export async function addWatchlistItem(listingID: string) {
  const res = await api.post(`/watchlist/${listingID}`);
  return res.data?.data;
}

export async function removeWatchlistItem(listingID: string) {
  const res = await api.delete(`/watchlist/${listingID}`);
  return res.data?.data;
}

export function setWatchlistPriceSnapshot(listingID: string, price: number) {
  const map = getWatchlistPriceMap();
  map[listingID] = price;
  setWatchlistPriceMap(map);
}

export function removeWatchlistPriceSnapshot(listingID: string) {
  const map = getWatchlistPriceMap();
  delete map[listingID];
  setWatchlistPriceMap(map);
}

export function getWatchlistPriceSnapshot(listingID: string): number | null {
  const map = getWatchlistPriceMap();
  const value = map[listingID];
  return typeof value === "number" ? value : null;
}

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("access_token");
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

let _isRefreshing = false;
let _refreshQueue: Array<(token: string) => void> = [];

function processQueue(newToken: string) {
  _refreshQueue.forEach((cb) => cb(newToken));
  _refreshQueue = [];
}

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;

    // Only attempt refresh on 401, once per request, and only if we have a refresh token
    if (
      error.response?.status === 401 &&
      !original._retry &&
      typeof window !== "undefined" &&
      localStorage.getItem("refresh_token")
    ) {
      original._retry = true;

      if (_isRefreshing) {
        // Queue this request until the refresh completes
        return new Promise((resolve) => {
          _refreshQueue.push((token: string) => {
            original.headers.Authorization = `Bearer ${token}`;
            resolve(api(original));
          });
        });
      }

      _isRefreshing = true;
      try {
        const { data } = await axios.post(`${BASE_URL}/auth/refresh`, {
          refresh_token: localStorage.getItem("refresh_token"),
        });
        const { access_token, refresh_token } = data.data as {
          access_token: string;
          refresh_token: string;
        };

        localStorage.setItem("access_token", access_token);
        localStorage.setItem("refresh_token", refresh_token);

        api.defaults.headers.common.Authorization = `Bearer ${access_token}`;
        original.headers.Authorization = `Bearer ${access_token}`;

        processQueue(access_token);
        return api(original);
      } catch {
        // Refresh failed — clear session and redirect to login
        localStorage.removeItem("access_token");
        localStorage.removeItem("refresh_token");
        localStorage.removeItem("auth_user");
        processQueue("");
        if (typeof window !== "undefined") {
          window.location.href = "/login";
        }
        return Promise.reject(error);
      } finally {
        _isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default api;
