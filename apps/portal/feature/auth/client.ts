import { BaseClient } from "@/lib/api/base-client";
import type { CommonSuccessResponse } from "@/feature/common/type";

export type LoginRequest = {
  username: string;
  password: string;
};

export type LoginResponse = {
  access_token: string;
  expires_in: number;
  token_type: string;
};

export type RegisterRequest = {
  username: string;
  email: string;
  password: string;
};

export type RegisterResponse = {
  id: number;
  username: string;
  email: string;
  created_at: string;
};

export type ProfileResponse = {
  id: number;
  username: string;
  email: string;
  created_at: string;
};

export class AuthClient extends BaseClient {
  login(body: LoginRequest) {
    return this.post<CommonSuccessResponse<LoginResponse>>({
      endpoint: "/auth/login",
      body,
    });
  }

  register(body: RegisterRequest) {
    return this.post<CommonSuccessResponse<RegisterResponse>>({
      endpoint: "/auth/register",
      body,
    });
  }

  getProfile() {
    return this.get<CommonSuccessResponse<ProfileResponse>>({
      endpoint: "/auth/profile",
    });
  }
}

export const authClient = new AuthClient();
