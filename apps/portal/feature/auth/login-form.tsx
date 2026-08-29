"use client";

import { useActionState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";
import type { AppLocale } from "@/i18n/routing";
import { loginAction } from "./actions";
import { initialAuthFormState } from "./types";

export function LoginForm({ locale }: { locale: AppLocale }) {
  const t = useTranslations("Auth");
  const [state, formAction, isPending] = useActionState(
    loginAction.bind(null, locale),
    initialAuthFormState,
  );

  return (
    <form action={formAction} className="flex w-full max-w-sm flex-col gap-4">
      <div className="flex flex-col gap-1">
        <label htmlFor="username" className="text-sm font-medium">
          {t("username")}
        </label>
        <input
          id="username"
          name="username"
          type="text"
          required
          autoComplete="username"
          className="rounded border border-black/[.08] bg-transparent px-3 py-2 dark:border-white/[.145]"
        />
        {state.fieldErrors?.username && (
          <p className="text-sm text-red-600">{state.fieldErrors.username}</p>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="password" className="text-sm font-medium">
          {t("password")}
        </label>
        <input
          id="password"
          name="password"
          type="password"
          required
          autoComplete="current-password"
          className="rounded border border-black/[.08] bg-transparent px-3 py-2 dark:border-white/[.145]"
        />
        {state.fieldErrors?.password && (
          <p className="text-sm text-red-600">{state.fieldErrors.password}</p>
        )}
      </div>

      {state.status === "error" && state.message && (
        <p className="text-sm text-red-600">{state.message}</p>
      )}

      <button
        type="submit"
        disabled={isPending}
        className="rounded-full bg-foreground px-5 py-2 font-medium text-background transition-colors hover:bg-[#383838] disabled:opacity-60 dark:hover:bg-[#ccc]"
      >
        {isPending ? t("loggingIn") : t("login")}
      </button>

      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("noAccount")}{" "}
        <Link
          href="/register"
          className="font-medium text-zinc-950 dark:text-zinc-50"
        >
          {t("createAccount")}
        </Link>
      </p>
    </form>
  );
}
