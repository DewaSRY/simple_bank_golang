import type { AxiosResponse } from "axios";

export type ApiMeta = {
  page: number;
  limit: number;
  total: number;
};

export type CommonSuccessResponse<T = unknown> = {
  message?: string;
  meta?: ApiMeta;
  data: T;
};

export interface ErrorDetail {
  field: string;
  message: string;
}

export interface ErrorResponse {
  error: {
    code: string;
    details: ErrorDetail[];
    message: string;
  };
}

export type AxiosResponseWrapper<T = unknown> = AxiosResponse<
  CommonSuccessResponse<T>
>;
