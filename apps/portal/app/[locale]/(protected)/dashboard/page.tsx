import { notFound } from "next/navigation";
import {
  QueryClient,
  HydrationBoundary,
  dehydrate,
} from "@tanstack/react-query";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { AccountList } from "@/components/dashboard/account-list";

interface props extends PageProps<"/[locale]/dashboard"> {
  searchParams: Promise<{ search?: string }>;
}

export default async function DashboardPage({ params }: props) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  const { t } = await getTranslation(locale, "common");

  const queryClient = new QueryClient();

  return (
    <div className="flex flex-1 flex-col bg-zinc-50 px-4 py-6 font-sans dark:bg-black sm:px-6">
      <div className="mx-auto w-full max-w-6xl">
        <div className="flex w-full items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">
              {t("dashboardTitle")}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("yourAccounts")}
            </p>
          </div>
        </div>

        <div className="mt-6">
          <HydrationBoundary state={dehydrate(queryClient)}>
            <AccountList />
          </HydrationBoundary>
        </div>
      </div>
    </div>
  );
}
