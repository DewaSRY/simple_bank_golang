import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from "axios";
import { ApiInterceptor } from "./api-interceptor";

export type SafeParamValue = string | number | boolean | string[];

export interface RequestParams {
  [key: string]: SafeParamValue;
}

export const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:3000",
  headers: {
    "Content-Type": "application/json",
  },
  timeout: 10_000,
});

export interface RequestOptions {
  endpoint: string;
  body?: unknown;
  params?: RequestParams;
  config?: AxiosRequestConfig;
}

export interface GetRequestOptions {
  endpoint: string;
  params?: RequestParams;
  config?: AxiosRequestConfig;
}

const interceptedInstances = new WeakSet<AxiosInstance>();

export class BaseClient {
  protected readonly instance: AxiosInstance;

  constructor(instance: AxiosInstance = apiClient) {
    this.instance = instance;

    if (!interceptedInstances.has(instance)) {
      new ApiInterceptor(instance);
      interceptedInstances.add(instance);
    }
  }

  protected get<TResponse = unknown>(
    options: GetRequestOptions,
  ): Promise<AxiosResponse<TResponse>> {
    const { endpoint, params, config = {} } = options;

    return this.instance.get<TResponse>(endpoint, {
      ...config,
      params,
    });
  }

  protected post<TResponse = unknown>(
    options: RequestOptions,
  ): Promise<AxiosResponse<TResponse>> {
    const { endpoint, body, params, config = {} } = options;

    return this.instance.post<TResponse>(endpoint, body, {
      ...config,
      params,
    });
  }

  protected put<TResponse = unknown>(
    options: RequestOptions,
  ): Promise<AxiosResponse<TResponse>> {
    const { endpoint, body, params, config = {} } = options;

    return this.instance.put<TResponse>(endpoint, body, {
      ...config,
      params,
    });
  }

  protected patch<TResponse = unknown>(
    options: RequestOptions,
  ): Promise<AxiosResponse<TResponse>> {
    const { endpoint, body, params, config = {} } = options;

    return this.instance.patch<TResponse>(endpoint, body, {
      ...config,
      params,
    });
  }

  protected delete<TResponse = unknown>(
    options: Omit<RequestOptions, "body">,
  ): Promise<AxiosResponse<TResponse>> {
    const { endpoint, params, config = {} } = options;

    return this.instance.delete<TResponse>(endpoint, {
      ...config,
      params,
    });
  }
}
