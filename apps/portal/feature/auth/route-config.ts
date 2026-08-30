/**
 * Single source of truth for which paths require an authenticated session
 * (`(protected)` route group) and which are only meant for signed-out users
 * (`(public)` route group's login/register). Paths are locale-stripped,
 * e.g. `/en/dashboard` -> `/dashboard`.
 */
export const PROTECTED_PATH_PREFIXES = ["/dashboard"];
export const AUTH_ONLY_PATHS = ["/login", "/register"];
