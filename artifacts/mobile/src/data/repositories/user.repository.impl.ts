import type {
  UpdateProfilePayload,
  UserRepository,
} from "../../domain/repositories/user.repository";
import type { PublicUser, User } from "../../domain/entities";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { USER_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope } from "../api/types";
import {
  toPublicUserDto,
  toUser,
  type ApiPublicUser,
  type ApiUser,
} from "../mappers/user.mapper";

export class HttpUserRepository implements UserRepository {
  constructor(private readonly http: HttpClient) {}

  async getPublicProfile(userId: string): Promise<PublicUser> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiPublicUser>>(
      USER_ENDPOINTS.publicProfile(userId),
    );
    return toPublicUserDto(unwrapEnvelope(data));
  }

  async updateMe(payload: UpdateProfilePayload): Promise<User> {
    const { data } = await this.http.instance.patch<ApiEnvelope<ApiUser>>(
      USER_ENDPOINTS.me,
      payload,
    );
    return toUser(unwrapEnvelope(data));
  }
}
