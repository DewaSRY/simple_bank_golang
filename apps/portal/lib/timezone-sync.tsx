"use client";

import { useEffect } from "react";
import { TIMEZONE_COOKIE_NAME } from "@/lib/api/constants";

// Writes the visitor's IANA timezone to a (non-httpOnly, non-sensitive)
// cookie so lib/api/api-interceptor.ts can attach it as `X-Timezone` on
// every server-side request — see docs/MIGRATION_TO_FULL_SSR.md Phase 4.
// Renders nothing; it only needs to run once per browser/timezone change.
export function TimezoneSync() {
  useEffect(() => {
    try {
      const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      const match = document.cookie.match(
        new RegExp(`(?:^|;\\s*)${TIMEZONE_COOKIE_NAME}=([^;]*)`),
      );
      const current = match ? decodeURIComponent(match[1]) : "";

      if (current !== timezone) {
        document.cookie = `${TIMEZONE_COOKIE_NAME}=${encodeURIComponent(
          timezone,
        )}; path=/; max-age=31536000; SameSite=Lax`;
      }
    } catch {
      // Best-effort only — a missing timezone header just means the backend
      // falls back to its own default.
    }
  }, []);

  return null;
}
