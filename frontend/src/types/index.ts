export interface User {
  id: string
  name: string
  email: string
  avatar_url?: string
  bio?: string
  location?: string
  rating: number
  review_count: number
  is_verified: boolean
  created_at: string
}

export interface Category {
  id: string
  name_en: string
  name_ar: string
  slug: string
  icon: string
  children?: Category[]
}

export interface ListingImage {
  id: string
  url: string
  is_cover: boolean
  sort_order: number
}

export interface Listing {
  id: string
  user_id: string
  category_id: string
  title: string
  description: string
  price?: number
  currency: string
  price_type: 'fixed' | 'negotiable' | 'free' | 'contact'
  condition: 'new' | 'used' | 'refurbished'
  status: 'active' | 'sold' | 'expired' | 'draft'
  type: 'sell' | 'buy' | 'rent' | 'auction' | 'service'
  country: string
  city: string
  address?: string
  latitude?: number
  longitude?: number
  view_count: number
  favorite_count: number
  is_featured: boolean
  created_at: string
  images?: ListingImage[]
  category?: Category
}

export interface Bid {
  id: string
  auction_id: string
  user_id: string
  amount: number
  placed_at: string
}

export interface Auction {
  id: string
  listing_id: string
  seller_id: string
  start_price: number
  reserve_price?: number
  buy_now_price?: number
  current_bid: number
  bid_count: number
  winner_id?: string
  status: 'active' | 'ended' | 'cancelled' | 'sold'
  starts_at: string
  ends_at: string
  currency: string
  bids?: Bid[]
}

export interface Message {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  type: 'text' | 'image' | 'offer'
  read_at?: string
  created_at: string
}

export interface Conversation {
  id: string
  listing_id?: string
  last_message_at?: string
  members: { user_id: string }[]
  messages?: Message[]
}

export interface PaginationMeta {
  total: number
  page: number
  per_page: number
  pages: number
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
  meta?: PaginationMeta
}
