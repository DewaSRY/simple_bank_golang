import { z } from "zod";

type Translate = (key: string) => string;

export function createLoginSchema(t: Translate) {
  return z.object({
    email: z.string().min(1, t("emailRequired")).email(t("emailInvalid")),
    password: z.string().min(1, t("passwordRequired")),
  });
}

export type LoginFormValues = z.infer<ReturnType<typeof createLoginSchema>>;

export function createRegisterSchema(t: Translate) {
  return z
    .object({
      username: z.string().min(1, t("usernameRequired")),
      email: z.string().min(1, t("emailRequired")).email(t("emailInvalid")),
      password: z.string().min(8, t("passwordMinLength")),
      confirmPassword: z.string().min(1, t("confirmPasswordRequired")),
    })
    .refine((data) => data.password === data.confirmPassword, {
      message: t("passwordMismatch"),
      path: ["confirmPassword"],
    });
}

export type RegisterFormValues = z.infer<
  ReturnType<typeof createRegisterSchema>
>;
