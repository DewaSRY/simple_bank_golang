import { QueryClient } from "@tanstack/react-query";
import axios from "axios";

const MAX_QUERY_RETRIES = 2;

/**
 * Retries network errors and 5xx responses (transient) but never 4xx
 * (auth/validation failures that a retry can't fix).
 */
function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_QUERY_RETRIES) return false;
  if (axios.isAxiosError(error)) {
    const status = error.response?.status;
    return status === undefined || status >= 500;
  }
  return false;
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 5 * 60_000,
        retry: shouldRetryQuery,
      },
      mutations: {
        // Mutations are user-triggered writes (login, create, update) —
        // auto-retrying could resubmit a form the user already resubmits
        // themselves via the button, so retries are left to the caller.
        retry: false,
      },
    },
  });
}
