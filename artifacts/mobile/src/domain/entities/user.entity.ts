export type UserRole = "shopper" | "traveler" | "seller" | "buyer" | "admin";

export interface User {
  readonly id: string;
  readonly name: string;
  readonly email: string;
  readonly phone?: string;
  readonly avatar?: string;
  readonly location?: string;
  readonly bio?: string;
  readonly rating: number;
  readonly totalSales: number;
  readonly balance: number;
  readonly isVerified: boolean;
  readonly roles: ReadonlyArray<UserRole>;
  readonly joinedAt: string;
}

export interface PublicUser {
  readonly id: string;
  readonly name: string;
  readonly avatar?: string;
  readonly rating: number;
  readonly isVerified: boolean;
}

export function toPublicUser(user: User): PublicUser {
  return {
    id: user.id,
    name: user.name,
    avatar: user.avatar,
    rating: user.rating,
    isVerified: user.isVerified,
  };
}

export function hasRole(user: User, role: UserRole): boolean {
  return user.roles.includes(role);
}
