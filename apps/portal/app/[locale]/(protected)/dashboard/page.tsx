import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import {
  QueryClient,
  HydrationBoundary,
  dehydrate,
} from "@tanstack/react-query";
import { routing } from "@/i18n/routing";
import { LogoutButton } from "@/components/auth/logout-button";
import { AccountList } from "@/feature/account/account-list";
import { accountKeys, fetchAccounts } from "@/feature/account/hooks/query";

export default async function DashboardPage({
  params,
}: PageProps<"/[locale]/dashboard">) {
  const { locale } = await params;

  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }

  setRequestLocale(locale);

  const t = await getTranslations("Common");

  const accountParams = { page: 1, limit: 10 };
  const queryClient = new QueryClient();
  await queryClient.prefetchQuery({
    queryKey: accountKeys.list(accountParams),
    queryFn: () => fetchAccounts(accountParams),
  });

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

        <HydrationBoundary state={dehydrate(queryClient)}>
          <AccountList />
        </HydrationBoundary>
      </div>
    </div>
  );
}
