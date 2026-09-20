import axios from "axios";
import type { ApiErrorResponse } from "./types";

// Next.js redacts a Server Action's thrown error in production (see
// docs/MIGRATION_TO_FULL_SSR.md and node_modules/next/dist/docs/01-app/
// 01-getting-started/10-error-handling.md: "avoid try/catch ... model
// expected errors as return values"). A raw AxiosError thrown from an
// action would arrive on the client as a generic, stripped Error, so
// lib/api/error.ts's getApiErrorMessage/getApiFieldErrors would never see
// `response.data.error`. Every feature action wraps its call in
// runServerAction so an expected API failure travels back as a normal
// (serializable) return value instead, and the client-side hook rethrows it
// via unwrapActionResult — reconstructing an axios-like error so every
// existing onError/getApiFieldErrors call site keeps working unchanged.
export type ActionErrorPayload = {
  status?: number;
  data?: ApiErrorResponse;
};

export type ActionResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: ActionErrorPayload };

export async function runServerAction<T>(
  fn: () => Promise<T>,
): Promise<ActionResult<T>> {
  try {
    return { ok: true, data: await fn() };
  } catch (error) {
    if (axios.isAxiosError<ApiErrorResponse>(error) && error.response) {
      return {
        ok: false,
        error: { status: error.response.status, data: error.response.data },
      };
    }

    // Not a "shaped" API error (network failure, a redirect()/notFound()
    // control-flow throw, BuildPhaseSkippedError, a real bug) — let it
    // propagate and go through Next's normal handling instead of forcing it
    // into a shape getApiErrorMessage can't parse anyway.
    throw error;
  }
}

export function unwrapActionResult<T>(result: ActionResult<T>): T {
  if (result.ok) {
    return result.data;
  }

  const error = new Error(
    result.error.data?.error?.message ?? "Request failed",
  ) as Error & {
    isAxiosError: true;
    response: ActionErrorPayload;
  };
  error.isAxiosError = true;
  error.response = result.error;
  throw error;
}
