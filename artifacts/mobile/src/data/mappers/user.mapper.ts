import type { PublicUser, User, UserRole } from "../../domain/entities";

export interface ApiUser {
  id: string;
  name: string;
  email: string;
  phone?: string | null;
  avatar_url?: string | null;
  avatar?: string | null;
  location?: string | null;
  bio?: string | null;
  rating?: number | null;
  total_sales?: number | null;
  balance?: number | null;
  is_verified?: boolean | null;
  roles?: string[] | null;
  role?: string | null;
  created_at?: string | null;
  joined_at?: string | null;
}

function normaliseRoles(
  api: Pick<ApiUser, "roles" | "role">,
): ReadonlyArray<UserRole> {
  const raw = api.roles ?? (api.role ? [api.role] : []);
  const valid: UserRole[] = ["shopper", "traveler", "seller", "buyer", "admin"];
  const set = new Set<UserRole>();
  for (const r of raw) {
    const candidate = String(r).toLowerCase() as UserRole;
    if (valid.includes(candidate)) set.add(candidate);
  }
  if (set.size === 0) set.add("buyer");
  return Array.from(set);
}

export function toUser(api: ApiUser): User {
  return {
    id: api.id,
    name: api.name,
    email: api.email,
    phone: api.phone ?? undefined,
    avatar: api.avatar_url ?? api.avatar ?? undefined,
    location: api.location ?? undefined,
    bio: api.bio ?? undefined,
    rating: api.rating ?? 0,
    totalSales: api.total_sales ?? 0,
    balance: api.balance ?? 0,
    isVerified: Boolean(api.is_verified),
    roles: normaliseRoles(api),
    joinedAt: api.joined_at ?? api.created_at ?? new Date().toISOString(),
  };
}

export interface ApiPublicUser {
  id: string;
  name: string;
  avatar_url?: string | null;
  avatar?: string | null;
  rating?: number | null;
  is_verified?: boolean | null;
}

export function toPublicUserDto(api: ApiPublicUser): PublicUser {
  return {
    id: api.id,
    name: api.name,
    avatar: api.avatar_url ?? api.avatar ?? undefined,
    rating: api.rating ?? 0,
    isVerified: Boolean(api.is_verified),
  };
}
