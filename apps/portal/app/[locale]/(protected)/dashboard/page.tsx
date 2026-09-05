import { notFound } from "next/navigation";
import {
  QueryClient,
  HydrationBoundary,
  dehydrate,
} from "@tanstack/react-query";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { AccountList } from "@/components/dashboard/account-list";
import { accountKeys, fetchAccounts } from "@/feature/account/hooks/query";

interface props extends PageProps<"/[locale]/dashboard"> {
  searchParams: Promise<{ search?: string }>;
}

export default async function DashboardPage({ params }: props) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  const { t } = await getTranslation(locale, "common");

  const accountParams = { page: 1, limit: 10 };
  const queryClient = new QueryClient();

  await queryClient.prefetchQuery({
    queryKey: accountKeys.list(accountParams),
    queryFn: () => fetchAccounts(accountParams),
  });

  return (
    <div className="flex my-2 flex-1 flex-col bg-zinc-50 px-6 font-sans dark:bg-black">
      <div className="mx-auto flex w-full  items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("dashboardTitle")}
        </h1>
      </div>

      <div className=" mt-4">
        <HydrationBoundary state={dehydrate(queryClient)}>
          {/* <AccountList /> */}
        </HydrationBoundary>
      </div>
    </div>
  );
}
