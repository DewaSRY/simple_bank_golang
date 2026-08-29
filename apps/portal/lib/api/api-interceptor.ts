import { AxiosInstance, InternalAxiosRequestConfig } from "axios";

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
        token = cookieStore.get("session_token")?.value || "";
      } catch (error) {
        // Next bails routes out of static generation by throwing here when cookies()
        if ((error as { digest?: string })?.digest === "DYNAMIC_SERVER_USAGE") {
          throw error;
        }
      }
    } else {
      const match = document.cookie.match(/(?:^|;\s*)session_token=([^;]*)/);
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
}
