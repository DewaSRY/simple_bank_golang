import { notFound } from "next/navigation";
import { connection } from "next/server";
import {
  QueryClient,
  HydrationBoundary,
  dehydrate,
} from "@tanstack/react-query";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { AccountList } from "@/feature/account/components/account-list";
import { queryKeys, listMeAccountsAction } from "@/feature/account";
import { unpackActionResult } from "@/lib/api/unpack-server-result";
import { PageContainer, PageHeader } from "@/components/common/page-header";
import { DashboardQuickActions } from "@/components/navigation/quick-actions";

interface props extends PageProps<"/[locale]/dashboard"> {
  searchParams: Promise<{ search?: string }>;
}

export default async function DashboardPage({ params }: props) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  const { t } = await getTranslation(locale, "common");

  // Per-user data: opt out of build-time prerendering so the prefetch below
  // only runs at request time (see BuildPhaseSkippedError).
  await connection();

  const queryClient = new QueryClient();

  // Matches AccountList's own useAccounts({ page: 1, limit: 10, name: "" })
  // call exactly (including the empty `name`), since TanStack Query hashes
  // the params object into the cache key.
  const query = { page: 1, limit: 10, name: "" };
  await queryClient.query({
    queryKey: queryKeys.list(query),
    queryFn: () => listMeAccountsAction(query).then(unpackActionResult),
  });

  return (
    <PageContainer>
      <PageHeader
        title={t("dashboardTitle")}
        description={t("dashboardDescription")}
        actions={<DashboardQuickActions />}
      />

      <HydrationBoundary state={dehydrate(queryClient)}>
        <AccountList />
      </HydrationBoundary>
    </PageContainer>
  );
}
