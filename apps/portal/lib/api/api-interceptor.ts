import { AxiosInstance, InternalAxiosRequestConfig } from "axios";
import { SESSION_COOKIE_NAME } from "@/feature/auth/constants";
// import logger from "@/lib/logger";

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

      await this.addAuthorizationHeader(config);
      this.addClientTimezoneHeader(config);
      return config;
    });
  }

  private async addAuthorizationHeader(
    config: InternalAxiosRequestConfig,
  ): Promise<void> {
    let token = "";

    if (this.isServer()) {
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
    } else {
      const match = document.cookie.match(
        new RegExp(`(?:^|;\\s*)${SESSION_COOKIE_NAME}=([^;]*)`),
      );
      token = match ? decodeURIComponent(match[1]) : "";
    }

    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }

  private addClientTimezoneHeader(config: InternalAxiosRequestConfig): void {
    if (this.isServer()) return;
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    config.headers["X-Timezone"] = timezone;
  }

  private setupResponseInterceptors(): void {
    this.instance.interceptors.response.use(
      async (response) => {
        if (typeof window === "undefined") {
          const { default: logger } = await import("@/lib/logger");
          logger.info(`API Request Success`, {
            method: response.config?.method?.toUpperCase(),
            path: response.config?.url,
            status: response.status,
            request_body: response.config?.data,
            response_data: response.data,
            request_params: response.config?.params,
            response_header: response.headers,
            request_header: response.config?.headers,
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
        if (status === 401) {
          await this.handleUnauthorized();
        }

        if (typeof window === "undefined") {
          const { default: logger } = await import("@/lib/logger");

          logger.error(`API Request Failed`, {
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
    console.warn("🔒 Unauthorized access detected. Redirecting to logout...");

    if (!this.isServer()) {
      window.location.href = "/logout";
    }
  }
}
