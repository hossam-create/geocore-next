export type TripStatus =
  | "active"
  | "matched"
  | "in_transit"
  | "completed"
  | "cancelled";

export type DeliveryStatus =
  | "pending"
  | "matched"
  | "accepted"
  | "locked"
  | "picked_up"
  | "in_transit"
  | "delivered"
  | "cancelled"
  | "disputed";

export interface Trip {
  id: string;
  traveler_id: string;
  origin_country: string;
  origin_city: string;
  origin_address?: string;
  dest_country: string;
  dest_city: string;
  dest_address?: string;
  departure_date: string;
  arrival_date: string;
  available_weight: number;
  max_items: number;
  price_per_kg: number;
  base_price: number;
  currency: string;
  notes?: string;
  frequency: string;
  status: TripStatus;
  created_at: string;
  updated_at: string;
}

export interface DeliveryRequest {
  id: string;
  buyer_id: string;
  trip_id?: string | null;
  traveler_id?: string | null;
  item_name: string;
  item_description?: string;
  item_url?: string;
  item_price: number;
  item_weight?: number | null;
  pickup_country: string;
  pickup_city: string;
  delivery_country: string;
  delivery_city: string;
  reward: number;
  currency: string;
  delivery_type: string;
  deadline?: string | null;
  notes?: string;
  status: DeliveryStatus;
  created_at: string;
  updated_at: string;
}

export interface MatchResult {
  trip_id: string;
  traveler_id: string;
  score: number;
  origin_country?: string;
  origin_city?: string;
  dest_country?: string;
  dest_city?: string;
  departure_date?: string;
  arrival_date?: string;
  available_weight?: number;
  price_per_kg?: number;
  base_price?: number;
  currency?: string;
}

export interface CreateTripPayload {
  origin_country: string;
  origin_city: string;
  origin_address?: string;
  dest_country: string;
  dest_city: string;
  dest_address?: string;
  departure_date: string;
  arrival_date: string;
  available_weight?: number;
  max_items?: number;
  price_per_kg?: number;
  base_price?: number;
  currency?: string;
  notes?: string;
  frequency?: string;
}

export interface CreateDeliveryRequestPayload {
  item_name: string;
  item_description?: string;
  item_url?: string;
  item_price: number;
  item_weight?: number;
  pickup_country: string;
  pickup_city: string;
  delivery_country: string;
  delivery_city: string;
  reward: number;
  currency?: string;
  delivery_type?: string;
  deadline?: string;
  notes?: string;
}
