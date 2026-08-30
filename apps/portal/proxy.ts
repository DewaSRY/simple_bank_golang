import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import createMiddleware from "next-intl/middleware";
import { routing, type AppLocale } from "./i18n/routing";
import { SESSION_COOKIE_NAME } from "./feature/auth/constants";
import {
  AUTH_ONLY_PATHS,
  PROTECTED_PATH_PREFIXES,
} from "./feature/auth/route-config";

const intlMiddleware = createMiddleware(routing);

function splitLocale(pathname: string): {
  locale: AppLocale | null;
  path: string;
} {
  for (const locale of routing.locales) {
    if (pathname === `/${locale}`) {
      return { locale, path: "/" };
    }
    if (pathname.startsWith(`/${locale}/`)) {
      return { locale, path: pathname.slice(`/${locale}`.length) };
    }
  }
  return { locale: null, path: pathname };
}

export default function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const { locale, path } = splitLocale(pathname);
  const isAuthenticated = Boolean(
    request.cookies.get(SESSION_COOKIE_NAME)?.value,
  );
  const activeLocale = locale ?? routing.defaultLocale;

  if (
    !isAuthenticated &&
    PROTECTED_PATH_PREFIXES.some((p) => path.startsWith(p))
  ) {
    const url = request.nextUrl.clone();
    url.pathname = `/${activeLocale}/login`;
    return NextResponse.redirect(url);
  }

  if (isAuthenticated && AUTH_ONLY_PATHS.includes(path)) {
    const url = request.nextUrl.clone();
    url.pathname = `/${activeLocale}/dashboard`;
    return NextResponse.redirect(url);
  }

  return intlMiddleware(request);
}

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
