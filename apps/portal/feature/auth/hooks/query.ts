import { useMutation, useQuery } from "@tanstack/react-query";
import { authClient } from "@/feature/auth/client";
import type {
  LoginRequest,
  AuthResponse,
  ProfileResponse,
  RegisterRequest,
} from "@/feature/auth/client";
import type { CommonSuccessResponse } from "@/feature/common/type";
import { logoutAction } from "../actions";
import type { AppLocale } from "@/i18n/settings";

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
  return useMutation<CommonSuccessResponse<AuthResponse>, Error, LoginRequest>({
    mutationFn: (body) =>
      authClient.login(body).then((response) => response.data),
  });
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

/**
 * Logout. `logoutAction` is a Server Action, so calling it here still runs
 * `cookies().delete()` server-side — the mutation just triggers that call
 * from client code instead of during a Server Component's render.
 */
export const useLogoutMutation = () => {
  return useMutation({
    mutationFn: (locale: AppLocale) => logoutAction(locale),
  });
};
