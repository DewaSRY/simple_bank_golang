import { AxiosInstance, InternalAxiosRequestConfig } from "axios";
import { SESSION_COOKIE_NAME } from "@/feature/auth/constants";
import { TIMEZONE_COOKIE_NAME } from "@/lib/api/constants";

declare module "axios" {
  export interface InternalAxiosRequestConfig {
    metadata?: {
      requestId: string;
      startTime: number;
    };
  }
}

function generateRequestId(): string {
  if (typeof globalThis.crypto?.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

export class BuildPhaseSkippedError extends Error {
  constructor(endpoint?: string) {
    super(
      `Skipped API request during "next build" static generation${endpoint ? `: ${endpoint}` : ""}`,
    );
    this.name = "BuildPhaseSkippedError";
  }
}

export class ApiInterceptor {
  constructor(private instance: AxiosInstance) {
    this.setupRequestInterceptors();
    this.setupResponseInterceptors();
  }

  private isServer(): boolean {
    return typeof window === "undefined";
  }

  private isBuildPhase(): boolean {
    return (
      this.isServer() && process.env.NEXT_PHASE === "phase-production-build"
    );
  }

  private setupRequestInterceptors(): void {
    this.instance.interceptors.request.use(async (config) => {
      if (this.isBuildPhase()) {
        throw new BuildPhaseSkippedError(config.url);
      }

      const requestId = generateRequestId();
      config.metadata = { requestId, startTime: Date.now() };
      config.headers["X-Request-Id"] = requestId;

      await this.addAuthorizationHeader(config);
      await this.addTimezoneHeader(config);

      if (typeof window === "undefined") {
        const { default: logger } = await import("@/lib/logger");
        logger.info(`API Request Started`, {
          requestId,
          method: config.method?.toUpperCase(),
          path: config.url,
          request_body: config.data,
          request_params: config.params,
          request_header: config.headers,
          ...(await this.getRequestDeviceInfo()),
        });
      }

      return config;
    });
  }

  // Every request now originates from a Server Action or Server Component
  // (docs/MIGRATION_TO_FULL_SSR.md Phases 1-3), so `isServer()` is always
  // true at this point — the document.cookie fallback this used to have for
  // browser-driven requests is gone (Phase 4).
  private async addAuthorizationHeader(
    config: InternalAxiosRequestConfig,
  ): Promise<void> {
    let token = "";

    try {
      const { cookies } = await import("next/headers");
      const cookieStore = await cookies();
      token = cookieStore.get(SESSION_COOKIE_NAME)?.value || "";
    } catch (error) {
      // Next bails routes out of static generation by throwing here when cookies()
      if ((error as { digest?: string })?.digest === "DYNAMIC_SERVER_USAGE") {
        throw error;
      }
    }

    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }

  // The timezone can no longer be read from `Intl`/`document` at request
  // time (the request runs on the server), so it's read back from the
  // cookie lib/timezone-sync.tsx writes on first client render.
  private async addTimezoneHeader(
    config: InternalAxiosRequestConfig,
  ): Promise<void> {
    try {
      const { cookies } = await import("next/headers");
      const cookieStore = await cookies();
      const timezone = cookieStore.get(TIMEZONE_COOKIE_NAME)?.value;
      if (timezone) {
        config.headers["X-Timezone"] = timezone;
      }
    } catch (error) {
      if ((error as { digest?: string })?.digest === "DYNAMIC_SERVER_USAGE") {
        throw error;
      }
    }
  }

  private setupResponseInterceptors(): void {
    this.instance.interceptors.response.use(
      async (response) => {
        if (typeof window === "undefined") {
          const { default: logger } = await import("@/lib/logger");
          const metadata = response.config?.metadata;
          logger.info(`API Request Success`, {
            requestId: metadata?.requestId,
            duration_ms: metadata ? Date.now() - metadata.startTime : undefined,
            method: response.config?.method?.toUpperCase(),
            path: response.config?.url,
            status: response.status,
            request_body: response.config?.data,
            response_data: response.data,
            request_params: response.config?.params,
            response_header: response.headers,
            request_header: response.config?.headers,
            response_data_length: response.headers.getAuthorization,
            ...(await this.getRequestDeviceInfo()),
          });
        }

        return response;
      },
      async (error) => {
        if (error instanceof BuildPhaseSkippedError) {
          return Promise.reject(error);
        }

        const status = error.response?.status;

        if (typeof window === "undefined") {
          const { default: logger } = await import("@/lib/logger");
          const metadata = error.config?.metadata;

          logger.error(`API Request Failed`, {
            requestId: metadata?.requestId,
            duration_ms: metadata ? Date.now() - metadata.startTime : undefined,
            method: error.config?.method?.toUpperCase(),
            path: error.config?.url,
            status: status ?? "network_error",
            message: error.response?.data?.message ?? error.message,
            request_body: error.config?.data,
            response_data: error.response?.data,
            request_params: error.config?.params,
            response_header: error.response?.headers,
            request_header: error.config?.headers,
            ...(await this.getRequestDeviceInfo()),
          });
        }

        // handleUnauthorized() throws on the server (next/navigation's
        // redirect()), so nothing below this runs for a 401 there.
        if (status === 401) {
          await this.handleUnauthorized();
        }

        return Promise.reject(error);
      },
    );
  }

  private async getRequestDeviceInfo(): Promise<{
    userAgent?: string;
    ip?: string;
    deviceType?: string;
  }> {
    try {
      const { headers } = await import("next/headers");
      const headersList = await headers();
      const userAgent = headersList.get("user-agent") ?? undefined;

      return {
        userAgent,
        ip:
          headersList.get("x-forwarded-for") ??
          headersList.get("x-real-ip") ??
          undefined,
        deviceType: this.getDeviceType(userAgent),
      };
    } catch {
      return {};
    }
  }

  private getDeviceType(userAgent?: string): string {
    if (!userAgent) return "unknown";
    if (/tablet|ipad/i.test(userAgent)) return "tablet";
    if (/mobi|android|iphone/i.test(userAgent)) return "mobile";
    return "desktop";
  }
  private async handleUnauthorized(): Promise<void> {
    if (this.isServer()) {
      const { redirect } = await import("@/i18n/redirect");
      redirect({ href: "/logout" });
      return;
    }

    window.location.href = "/logout";
  }
}
