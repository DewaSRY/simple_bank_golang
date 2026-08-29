import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { routing } from "@/i18n/routing";
import { accountClient, type Account } from "@/lib/api/clients/account-client";
import { getApiErrorMessage } from "@/lib/api/error";
import { LogoutButton } from "@/feature/auth/logout-button";

export default async function DashboardPage({
  params,
}: PageProps<"/[locale]/dashboard">) {
  const { locale } = await params;

  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }

  setRequestLocale(locale);

  const t = await getTranslations("Common");

  let accounts: Account[] = [];
  let loadError: string | null = null;
  try {
    const response = await accountClient.listAccounts({ page: 1, limit: 10 });
    accounts = response.data.data;
  } catch (error) {
    loadError = getApiErrorMessage(error, t("loadAccountsError"));
  }

  return (
    <div className="flex flex-1 flex-col gap-8 bg-zinc-50 px-6 py-16 font-sans dark:bg-black">
      <div className="mx-auto flex w-full max-w-2xl items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("dashboardTitle")}
        </h1>
        <LogoutButton locale={locale} />
      </div>

      <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
        <h2 className="text-lg font-medium">{t("yourAccounts")}</h2>

        {loadError && <p className="text-sm text-red-600">{loadError}</p>}

        {!loadError && accounts.length === 0 && (
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            {t("noAccounts")}
          </p>
        )}

        {accounts.length > 0 && (
          <ul className="flex flex-col gap-2">
            {accounts.map((account) => (
              <li
                key={account.id}
                className="flex items-center justify-between rounded border border-black/[.08] px-4 py-3 dark:border-white/[.145]"
              >
                <span>{account.owner}</span>
                <span className="font-mono">
                  {account.balance} {account.currency}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
