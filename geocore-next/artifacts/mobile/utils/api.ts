import axios from "axios";
import * as SecureStore from "expo-secure-store";

export const BASE_URL = "https://geo-core-next.replit.app/api/v1";

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use(async (config) => {
  try {
    const token = await SecureStore.getItemAsync("access_token");
    if (token) config.headers.Authorization = `Bearer ${token}`;
  } catch {}
  return config;
});

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401) {
      try {
        const refresh = await SecureStore.getItemAsync("refresh_token");
        if (refresh) {
          const { data } = await axios.post(`${BASE_URL}/auth/refresh`, {
            refresh_token: refresh,
          });
          const newToken = data?.data?.access_token;
          if (newToken) {
            await SecureStore.setItemAsync("access_token", newToken);
            error.config.headers.Authorization = `Bearer ${newToken}`;
            return api(error.config);
          }
        }
      } catch {
        await SecureStore.deleteItemAsync("access_token").catch(() => {});
        await SecureStore.deleteItemAsync("refresh_token").catch(() => {});
      }
    }
    return Promise.reject(error);
  }
);

export default api;

export const authAPI = {
  login: (email: string, password: string) =>
    api.post("/auth/login", { email, password }),
  register: (name: string, email: string, password: string, phone?: string) =>
    api.post("/auth/register", { name, email, password, phone }),
  logout: () => api.post("/auth/logout"),
  me: () => api.get("/users/me"),
};

export const listingsAPI = {
  getAll: (params?: {
    category?: string;
    sort?: string;
    page?: number;
    limit?: number;
  }) => api.get("/listings", { params }),
  getOne: (id: string) => api.get(`/listings/${id}`),
  create: (data: FormData) =>
    api.post("/listings", data, {
      headers: { "Content-Type": "multipart/form-data" },
    }),
};

export const bidsAPI = {
  placeBid: (listingId: string, amount: number) =>
    api.post(`/auctions/${listingId}/bids`, { amount }),
  getMyBids: () => api.get("/users/me/bids"),
};

export const messagesAPI = {
  getConversations: () => api.get("/chat/conversations"),
  getMessages: (conversationId: string) =>
    api.get(`/chat/conversations/${conversationId}/messages`),
  sendMessage: (conversationId: string, text: string) =>
    api.post(`/chat/conversations/${conversationId}/messages`, { text }),
};
