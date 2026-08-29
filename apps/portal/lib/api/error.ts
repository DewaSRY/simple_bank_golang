import axios from "axios";
import type { ApiErrorResponse } from "./types";

export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (
    axios.isAxiosError<ApiErrorResponse>(error) &&
    error.response?.data?.error?.message
  ) {
    return error.response.data.error.message;
  }

  return fallback;
}

export function getApiFieldErrors(
  error: unknown,
): Record<string, string> | undefined {
  if (
    axios.isAxiosError<ApiErrorResponse>(error) &&
    error.response?.data?.error?.details?.length
  ) {
    return Object.fromEntries(
      error.response.data.error.details.map((detail) => [
        detail.field,
        detail.message,
      ]),
    );
  }

  return undefined;
}
