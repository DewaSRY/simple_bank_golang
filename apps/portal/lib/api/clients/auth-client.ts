import { BaseClient } from "../base-client";
import type { ApiSuccessResponse } from "../types";

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  expires_in: number;
  token_type: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface RegisterResponse {
  id: number;
  username: string;
  email: string;
  created_at: string;
}

export class AuthClient extends BaseClient {
  login(body: LoginRequest) {
    return this.post<ApiSuccessResponse<LoginResponse>>({
      endpoint: "/auth/login",
      body,
    });
  }

  register(body: RegisterRequest) {
    return this.post<ApiSuccessResponse<RegisterResponse>>({
      endpoint: "/users",
      body,
    });
  }
}

export const authClient = new AuthClient();
