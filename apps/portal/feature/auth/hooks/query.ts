import { useMutation, useQuery } from "@tanstack/react-query";
import { authClient } from "@/feature/auth/client";
import type {
  LoginRequest,
  AuthResponse,
  ProfileResponse,
  RegisterRequest,
} from "@/feature/auth/client";
import type { CommonSuccessResponse } from "@/feature/common/type";

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
  return useMutation<CommonSuccessResponse<AuthResponse>, Error, LoginRequest>(
    {
      mutationFn: (body) =>
        authClient.login(body).then((response) => response.data),
    },
  );
};

/**
 * Register
 */
export const useRegisterMutation = () => {
  return useMutation<
    CommonSuccessResponse<AuthResponse>,
    Error,
    RegisterRequest
  >({
    mutationFn: (body) =>
      authClient.register(body).then((response) => response.data),
  });
};

/**
 * Get current authenticated user's profile.
 */
export const useProfileQuery = () => {
  return useQuery<CommonSuccessResponse<ProfileResponse>, Error>({
    queryKey: authQueryKeys.profile(),
    queryFn: () => authClient.getProfile().then((response) => response.data),
  });
};
