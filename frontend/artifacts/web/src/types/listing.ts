// Listing types for GeoCore Next

export interface ListingImage {
  id: string;
  url: string;
  thumbnail_url?: string;
  width?: number;
  height?: number;
}

export interface ListingLocation {
  city?: string;
  country?: string;
  address?: string;
  latitude?: number;
  longitude?: number;
}

export interface ListingCategory {
  id: string;
  name: string;
  slug: string;
}

export interface ListingAttributes {
  [key: string]: string | number | boolean | undefined;
}

export interface Listing {
  id: string;
  title: string;
  description?: string;
  price?: number;
  currency?: string;
  type?: 'fixed' | 'auction' | 'dutch' | 'reverse';
  condition?: 'new' | 'like-new' | 'good' | 'fair' | 'used';
  
  // Images
  images?: ListingImage[];
  image_url?: string;
  
  // Location
  city?: string;
  country?: string;
  location?: ListingLocation;
  
  // Category
  category?: ListingCategory;
  category_id?: string;
  
  // Auction fields
  is_auction?: boolean;
  isAuction?: boolean;
  start_price?: number;
  startPrice?: number;
  current_bid?: number;
  currentBid?: number;
  bid_count?: number;
  bids_count?: number;
  bidCount?: number;
  ends_at?: string;
  auctionEndsAt?: string;
  
  // Dutch auction
  auction_type?: string;
  auctionType?: string;
  clearing_price?: number;
  total_slots?: number;
  slots_won?: number;
  
  // Reverse auction
  lowest_offer?: number;
  offers?: AuctionOffer[];
  
  // Buy now
  buy_now_price?: number;
  buyNowPrice?: number;
  
  // Anti-sniping
  anti_sniping_enabled?: boolean;
  anti_sniping_extension_minutes?: number;
  
  // Flags
  is_featured?: boolean;
  isFeatured?: boolean;
  is_sold?: boolean;
  is_active?: boolean;
  
  // Attributes
  attributes?: ListingAttributes;
  
  // Timestamps
  created_at?: string;
  updated_at?: string;
}

export interface AuctionOffer {
  id: string;
  user_id: string;
  user_name?: string;
  amount: number;
  created_at: string;
}

export interface AuctionBid {
  id: string;
  user_id: string;
  user_name?: string;
  amount: number;
  created_at: string;
  is_winning?: boolean;
}

export interface Store {
  id: string;
  slug: string;
  name: string;
  description?: string;
  logo_url?: string;
  banner_url?: string;
  views?: number;
  rating?: number;
  created_at?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    total: number;
    page: number;
    per_page: number;
    pages: number;
  };
}

export type ListingsResponse = PaginatedResponse<Listing>;
export type StoresResponse = PaginatedResponse<Store>;
