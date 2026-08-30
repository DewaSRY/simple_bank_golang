import { useMutation, useQuery } from "@tanstack/react-query";
import { authClient } from "@/feature/auth/client";
import type {
  LoginRequest,
  AuthResponse,
  ProfileResponse,
  RegisterRequest,
} from "@/feature/auth/client";

import type { ApiResponse } from "@/feature/common/type";

/**
 * Centralized query keys for the auth feature.
 */
export const authQueryKeys = {
  all: ["auth"] as const,
  profile: () => [...authQueryKeys.all, "profile"] as const,
};

/**
 * Login
 */
export const useLoginMutation = () => {
  return useMutation<ApiResponse<AuthResponse>, Error, LoginRequest>({
    mutationFn: (body) => authClient.login(body),
  });
};

/**
 * Register
 */
export const useRegisterMutation = () => {
  return useMutation<ApiResponse<AuthResponse>, Error, RegisterRequest>({
    mutationFn: (body) => authClient.register(body),
  });
};

/**
 * Get current authenticated user's profile.
 */
export const useProfileQuery = () => {
  return useQuery<ApiResponse<ProfileResponse>, Error>({
    queryKey: authQueryKeys.profile(),
    queryFn: () => authClient.getProfile(),
  });
};
