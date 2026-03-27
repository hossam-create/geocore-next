import AsyncStorage from "@react-native-async-storage/async-storage";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { detectLocationByIP } from "@/utils/geo";

export type ListingCategory =
  | "all"
  | "vehicles"
  | "electronics"
  | "furniture"
  | "fashion"
  | "real-estate"
  | "services"
  | "sports";

export type ListingCondition = "new" | "like-new" | "good" | "fair" | "poor";

export type AuctionType = "standard" | "dutch" | "reverse";

export interface Listing {
  id: string;
  title: string;
  description: string;
  price: number;
  currency: string;
  category: ListingCategory;
  condition: ListingCondition;
  location: string;
  lat?: number;
  lon?: number;
  imageUrl?: string;
  sellerId: string;
  sellerName: string;
  sellerAvatar?: string;
  createdAt: string;
  views: number;
  isFavorited: boolean;
  isAuction: boolean;
  auctionType?: AuctionType;
  auctionEndsAt?: string;
  currentBid?: number;
  bidCount?: number;
  tags: string[];
  isFeatured?: boolean;
}

export interface Message {
  id: string;
  conversationId: string;
  senderId: string;
  text: string;
  createdAt: string;
  isRead: boolean;
}

export interface Conversation {
  id: string;
  listingId: string;
  listingTitle: string;
  listingImage?: string;
  otherUserId: string;
  otherUserName: string;
  otherUserAvatar?: string;
  lastMessage: string;
  lastMessageAt: string;
  unreadCount: number;
  messages: Message[];
}

export interface User {
  id: string;
  name: string;
  email: string;
  phone?: string;
  avatar?: string;
  location: string;
  joinedAt: string;
  rating: number;
  totalSales: number;
  bio: string;
  balance: number;
  isVerified: boolean;
}

interface AppContextType {
  user: User;
  listings: Listing[];
  conversations: Conversation[];
  favorites: string[];
  isLoggedIn: boolean;
  activeCurrency: string;
  setActiveCurrency: (code: string) => void;
  detectedLocation: string;
  addListing: (
    listing: Omit<Listing, "id" | "createdAt" | "views" | "isFavorited">
  ) => void;
  toggleFavorite: (listingId: string) => void;
  sendMessage: (conversationId: string, text: string) => void;
  startConversation: (listing: Listing) => string;
  markConversationRead: (conversationId: string) => void;
  totalUnread: number;
}

const MOCK_USER: User = {
  id: "user-1",
  name: "Ahmed Al-Rashid",
  email: "ahmed@example.com",
  phone: "+971501234567",
  location: "Dubai, UAE",
  joinedAt: "2023-06-15",
  rating: 4.8,
  totalSales: 47,
  bio: "Passionate collector and dealer of quality goods. Fast shipping, honest descriptions.",
  balance: 1250.0,
  isVerified: true,
};

const MOCK_LISTINGS: Listing[] = [
  {
    id: "l-1",
    title: "2021 BMW 3 Series – Immaculate Condition",
    description:
      "Full service history, single owner, no accidents. All original parts. Comes with 2 years warranty remaining.",
    price: 185000,
    currency: "AED",
    category: "vehicles",
    condition: "like-new",
    location: "Dubai, UAE",
    lat: 25.2048,
    lon: 55.2708,
    sellerId: "user-2",
    sellerName: "Khalid Motors",
    createdAt: new Date(Date.now() - 86400000 * 2).toISOString(),
    views: 342,
    isFavorited: false,
    isAuction: false,
    tags: ["bmw", "luxury", "sedan"],
    isFeatured: true,
  },
  {
    id: "l-2",
    title: "iPhone 15 Pro Max 256GB – Natural Titanium",
    description:
      "Purchased 3 months ago. Like new condition, always used with case. Original box and accessories included.",
    price: 4200,
    currency: "AED",
    category: "electronics",
    condition: "like-new",
    location: "Cairo, Egypt",
    sellerId: "user-3",
    sellerName: "TechDeals EG",
    createdAt: new Date(Date.now() - 86400000).toISOString(),
    views: 189,
    isFavorited: true,
    isAuction: false,
    tags: ["iphone", "apple", "smartphone"],
  },
  {
    id: "l-3",
    title: "Vintage Rolex Submariner 1968",
    description:
      "Rare vintage Submariner in collector grade condition. Papers and original bracelet intact. A true investment piece.",
    price: 0,
    currency: "AED",
    category: "fashion",
    condition: "good",
    location: "Riyadh, KSA",
    lat: 24.7136,
    lon: 46.6753,
    sellerId: "user-4",
    sellerName: "GulfLux",
    createdAt: new Date(Date.now() - 3600000 * 5).toISOString(),
    views: 521,
    isFavorited: false,
    isAuction: true,
    auctionType: "standard",
    auctionEndsAt: new Date(Date.now() + 86400000 * 2).toISOString(),
    currentBid: 28500,
    bidCount: 12,
    tags: ["rolex", "watch", "luxury", "vintage"],
    isFeatured: true,
  },
  {
    id: "l-4",
    title: "Herman Miller Aeron Chair – Size B",
    description:
      "Used in home office. In excellent working condition, all adjustments function perfectly. No tears or stains.",
    price: 2800,
    currency: "AED",
    category: "furniture",
    condition: "good",
    location: "Abu Dhabi, UAE",
    sellerId: "user-5",
    sellerName: "OfficeElite",
    createdAt: new Date(Date.now() - 86400000 * 4).toISOString(),
    views: 87,
    isFavorited: false,
    isAuction: false,
    tags: ["herman miller", "ergonomic", "office"],
  },
  {
    id: "l-5",
    title: "DJI Mavic 3 Pro Drone – Full Kit",
    description:
      "Complete kit with 3 batteries, carrying case, ND filter set. Only 12 hours total flight time. Perfect for photography.",
    price: 0,
    currency: "AED",
    category: "electronics",
    condition: "like-new",
    location: "Beirut, Lebanon",
    sellerId: "user-6",
    sellerName: "DroneHub",
    createdAt: new Date(Date.now() - 3600000 * 2).toISOString(),
    views: 203,
    isFavorited: false,
    isAuction: true,
    auctionType: "standard",
    auctionEndsAt: new Date(Date.now() + 86400000).toISOString(),
    currentBid: 5200,
    bidCount: 7,
    tags: ["dji", "drone", "photography"],
  },
  {
    id: "l-6",
    title: "Luxury 2BR Apartment – Downtown Views",
    description:
      "Stunning views of Burj Khalifa, fully furnished, premium amenities. Available for annual or short-term lease.",
    price: 180000,
    currency: "AED",
    category: "real-estate",
    condition: "new",
    location: "Downtown Dubai",
    lat: 25.1972,
    lon: 55.2744,
    sellerId: "user-7",
    sellerName: "PremiumProps",
    createdAt: new Date(Date.now() - 86400000 * 7).toISOString(),
    views: 1204,
    isFavorited: true,
    isAuction: false,
    tags: ["apartment", "dubai", "furnished", "luxury"],
    isFeatured: true,
  },
  {
    id: "l-7",
    title: "Trek Domane SL 6 Road Bike",
    description:
      "2022 model, carbon frame, Shimano Ultegra groupset. 1,200km ridden. Barely used, stored indoors.",
    price: 6500,
    currency: "AED",
    category: "sports",
    condition: "like-new",
    location: "Amman, Jordan",
    sellerId: "user-8",
    sellerName: "CyclePro",
    createdAt: new Date(Date.now() - 86400000 * 3).toISOString(),
    views: 98,
    isFavorited: false,
    isAuction: false,
    tags: ["trek", "road bike", "cycling"],
  },
  {
    id: "l-8",
    title: "Sony PlayStation 5 Console Bundle",
    description:
      "Disc edition PS5 with 3 controllers, charging dock, and 5 games. All in original packaging.",
    price: 0,
    currency: "AED",
    category: "electronics",
    condition: "good",
    location: "Kuwait City",
    sellerId: "user-9",
    sellerName: "GamingZone",
    createdAt: new Date(Date.now() - 3600000 * 8).toISOString(),
    views: 412,
    isFavorited: false,
    isAuction: true,
    auctionType: "reverse",
    auctionEndsAt: new Date(Date.now() + 86400000 * 3).toISOString(),
    currentBid: 850,
    bidCount: 23,
    tags: ["ps5", "playstation", "gaming"],
  },
];

const MOCK_CONVERSATIONS: Conversation[] = [
  {
    id: "conv-1",
    listingId: "l-2",
    listingTitle: "iPhone 15 Pro Max 256GB",
    otherUserId: "user-3",
    otherUserName: "TechDeals EG",
    lastMessage: "Is this still available? Can you do 3900?",
    lastMessageAt: new Date(Date.now() - 3600000).toISOString(),
    unreadCount: 2,
    messages: [
      {
        id: "m-1",
        conversationId: "conv-1",
        senderId: "user-1",
        text: "Hi, is this still available?",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
        isRead: true,
      },
      {
        id: "m-2",
        conversationId: "conv-1",
        senderId: "user-3",
        text: "Yes it is! Feel free to make an offer.",
        createdAt: new Date(Date.now() - 3700000).toISOString(),
        isRead: true,
      },
      {
        id: "m-3",
        conversationId: "conv-1",
        senderId: "user-1",
        text: "Is this still available? Can you do 3900?",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
        isRead: false,
      },
    ],
  },
  {
    id: "conv-2",
    listingId: "l-6",
    listingTitle: "Luxury 2BR Apartment",
    otherUserId: "user-7",
    otherUserName: "PremiumProps",
    lastMessage: "We can arrange a viewing this Thursday at 3pm.",
    lastMessageAt: new Date(Date.now() - 86400000).toISOString(),
    unreadCount: 0,
    messages: [
      {
        id: "m-4",
        conversationId: "conv-2",
        senderId: "user-1",
        text: "I am interested in the apartment. Is it still available?",
        createdAt: new Date(Date.now() - 86400000 * 2).toISOString(),
        isRead: true,
      },
      {
        id: "m-5",
        conversationId: "conv-2",
        senderId: "user-7",
        text: "We can arrange a viewing this Thursday at 3pm.",
        createdAt: new Date(Date.now() - 86400000).toISOString(),
        isRead: true,
      },
    ],
  },
];

const AppContext = createContext<AppContextType | null>(null);

export function AppProvider({ children }: { children: React.ReactNode }) {
  const [listings, setListings] = useState<Listing[]>(MOCK_LISTINGS);
  const [conversations, setConversations] =
    useState<Conversation[]>(MOCK_CONVERSATIONS);
  const [favorites, setFavorites] = useState<string[]>(
    MOCK_LISTINGS.filter((l) => l.isFavorited).map((l) => l.id)
  );
  const [activeCurrency, setActiveCurrencyState] = useState("AED");
  const [detectedLocation, setDetectedLocation] = useState("Dubai, UAE");

  const user = MOCK_USER;
  const isLoggedIn = true;

  useEffect(() => {
    const init = async () => {
      try {
        const savedFavs = await AsyncStorage.getItem("favorites");
        if (savedFavs) setFavorites(JSON.parse(savedFavs));

        const savedListings = await AsyncStorage.getItem("myListings");
        if (savedListings) {
          const myListings: Listing[] = JSON.parse(savedListings);
          setListings((prev) => {
            const existingIds = new Set(MOCK_LISTINGS.map((l) => l.id));
            const newOnes = myListings.filter((l) => !existingIds.has(l.id));
            return [...MOCK_LISTINGS, ...newOnes];
          });
        }

        const savedCurrency = await AsyncStorage.getItem("activeCurrency");
        if (savedCurrency) setActiveCurrencyState(savedCurrency);

        const ipInfo = await detectLocationByIP();
        if (ipInfo?.city && ipInfo?.country) {
          setDetectedLocation(`${ipInfo.city}, ${ipInfo.country}`);
        }
      } catch {}
    };
    init();
  }, []);

  const setActiveCurrency = useCallback(async (code: string) => {
    setActiveCurrencyState(code);
    await AsyncStorage.setItem("activeCurrency", code);
  }, []);

  const toggleFavorite = useCallback(
    async (listingId: string) => {
      const newFavs = favorites.includes(listingId)
        ? favorites.filter((id) => id !== listingId)
        : [...favorites, listingId];
      setFavorites(newFavs);
      await AsyncStorage.setItem("favorites", JSON.stringify(newFavs));
    },
    [favorites]
  );

  const addListing = useCallback(
    async (
      listingData: Omit<Listing, "id" | "createdAt" | "views" | "isFavorited">
    ) => {
      const newListing: Listing = {
        ...listingData,
        id: `l-${Date.now()}`,
        createdAt: new Date().toISOString(),
        views: 0,
        isFavorited: false,
      };
      setListings((prev) => [newListing, ...prev]);
      const myListings = listings.filter((l) => l.sellerId === user.id);
      await AsyncStorage.setItem(
        "myListings",
        JSON.stringify([...myListings, newListing])
      );
    },
    [listings, user.id]
  );

  const startConversation = useCallback(
    (listing: Listing) => {
      const existing = conversations.find(
        (c) =>
          c.listingId === listing.id && c.otherUserId === listing.sellerId
      );
      if (existing) return existing.id;

      const newConv: Conversation = {
        id: `conv-${Date.now()}`,
        listingId: listing.id,
        listingTitle: listing.title,
        otherUserId: listing.sellerId,
        otherUserName: listing.sellerName,
        lastMessage: "",
        lastMessageAt: new Date().toISOString(),
        unreadCount: 0,
        messages: [],
      };
      setConversations((prev) => [newConv, ...prev]);
      return newConv.id;
    },
    [conversations]
  );

  const sendMessage = useCallback(
    (conversationId: string, text: string) => {
      const msg: Message = {
        id: `m-${Date.now()}`,
        conversationId,
        senderId: user.id,
        text,
        createdAt: new Date().toISOString(),
        isRead: false,
      };
      setConversations((prev) =>
        prev.map((c) =>
          c.id === conversationId
            ? {
                ...c,
                messages: [...c.messages, msg],
                lastMessage: text,
                lastMessageAt: msg.createdAt,
              }
            : c
        )
      );
    },
    [user.id]
  );

  const markConversationRead = useCallback((conversationId: string) => {
    setConversations((prev) =>
      prev.map((c) =>
        c.id === conversationId
          ? {
              ...c,
              unreadCount: 0,
              messages: c.messages.map((m) => ({ ...m, isRead: true })),
            }
          : c
      )
    );
  }, []);

  const totalUnread = conversations.reduce((sum, c) => sum + c.unreadCount, 0);

  return (
    <AppContext.Provider
      value={{
        user,
        listings,
        conversations,
        favorites,
        isLoggedIn,
        activeCurrency,
        setActiveCurrency,
        detectedLocation,
        addListing,
        toggleFavorite,
        sendMessage,
        startConversation,
        markConversationRead,
        totalUnread,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}

export function useAppContext() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppContext must be used within AppProvider");
  return ctx;
}
