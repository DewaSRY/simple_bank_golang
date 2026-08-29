export interface AuthFormState {
  status: "idle" | "error";
  message?: string;
  fieldErrors?: Record<string, string>;
}

export const initialAuthFormState: AuthFormState = { status: "idle" };
