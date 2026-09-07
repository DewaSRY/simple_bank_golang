import { notFound } from "next/navigation";
import {
  QueryClient,
  HydrationBoundary,
  dehydrate,
} from "@tanstack/react-query";
import { isAppLocale } from "@/i18n/settings";

import { queryKeys as accountQueryKeys } from "@/feature/account-manage/hooks/query";
import { accountManageClient as accountApiClient } from "@/feature/account-manage/client";
import { queryKeys as accountTransactionQueryKeys } from "@/feature/account-transaction/hooks/query";
import { accountTransactionClient } from "@/feature/account-transaction/client";

import { AccountDetailView } from "@/components/account/account-detail-view";
import { ParamsSearchParams, parseIntParam } from "@/feature/common/params";

interface params extends PageProps<"/[locale]/account/[id]"> {
  searchParams: Promise<ParamsSearchParams>;
}

function getSearchParams(searchParams: ParamsSearchParams) {
  return {
    page: parseIntParam(searchParams.page, 1),
    limit: parseIntParam(searchParams.limit, 25),
  };
}

export default async function AccountDetailPage({
  params,
  searchParams,
}: params) {
  const { locale, id } = await params;
  const { page, limit } = getSearchParams(await searchParams);
  if (!isAppLocale(locale)) {
    notFound();
  }

  const accountId = Number(id);
  if (!Number.isInteger(accountId) || accountId <= 0) {
    notFound();
  }

  const queryClient = new QueryClient();

  await queryClient.prefetchQuery({
    queryKey: accountTransactionQueryKeys.entries(accountId, {
      limit: limit,
      page: page,
    }),
    queryFn: () =>
      accountTransactionClient
        .getAccountEntries(accountId, {
          limit: limit,
          page: page,
        })
        .then((res) => res.data),
  });

  await queryClient.prefetchQuery({
    queryKey: accountQueryKeys.manageAccount(accountId),
    queryFn: () =>
      accountApiClient.detailAccount(accountId).then((res) => res.data),
  });

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AccountDetailView accountId={accountId} />
    </HydrationBoundary>
  );
}
