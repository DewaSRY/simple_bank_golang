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
