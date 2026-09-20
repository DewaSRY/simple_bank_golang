import { useMutation, useQuery } from "@tanstack/react-query";
import type {
  LoginRequest,
  ProfileResponse,
  RegisterRequest,
} from "@/feature/auth/type";
import type { CommonSuccessResponse } from "@/feature/common";
import { unwrapActionResult } from "@/lib/api/action-result";
import {
  getProfileAction,
  loginAction,
  logoutAction,
  registerAction,
} from "../actions";
import type { AppLocale } from "@/i18n/settings";

/**
 * Centralized query keys for the auth feature.
 */
export const authQueryKeys = {
  all: ["auth"] as const,
  profile: () => [...authQueryKeys.all, "profile"] as const,
};

/**
 * Login. `loginAction` is a Server Action — it runs `authClient.login` and
 * sets the session cookie server-side, so the browser never talks to
 * ../core-service directly.
 */
export const useLoginMutation = () => {
  return useMutation<null, Error, LoginRequest>({
    mutationFn: (body) => loginAction(body).then(unwrapActionResult),
  });
};

/**
 * Register. Same Server Action shape as login.
 */
export const useRegisterMutation = () => {
  return useMutation<null, Error, RegisterRequest>({
    mutationFn: (body) => registerAction(body).then(unwrapActionResult),
  });
};

/**
 * Get current authenticated user's profile.
 */
export const useProfileQuery = () => {
  return useQuery<CommonSuccessResponse<ProfileResponse>, Error>({
    queryKey: authQueryKeys.profile(),
    queryFn: () => getProfileAction().then(unwrapActionResult),
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
