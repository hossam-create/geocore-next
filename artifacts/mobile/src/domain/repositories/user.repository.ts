import type { PublicUser, User } from "../entities";

export interface UpdateProfilePayload {
  readonly name?: string;
  readonly bio?: string;
  readonly location?: string;
  readonly avatar?: string;
  readonly phone?: string;
}

export interface UserRepository {
  getPublicProfile(userId: string): Promise<PublicUser>;
  updateMe(payload: UpdateProfilePayload): Promise<User>;
}
