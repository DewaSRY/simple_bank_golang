export type LoginRequest = {
  email: string;
  password: string;
};

export type AuthResponse = {
  access_token: string;
  expires_in: number;
  token_type: string;
};

export type RegisterRequest = {
  username: string;
  email: string;
  password: string;
  password_confirm: string;
};

export type ProfileResponse = {
  id: number;
  username: string;
  email: string;
  created_at: string;
};
