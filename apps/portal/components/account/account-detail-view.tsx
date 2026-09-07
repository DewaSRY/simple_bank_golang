"use client";

import { ArrowLeft } from "lucide-react";
import { Link } from "@/i18n/navigation";

import { useAccountDetail } from "@/feature/account-manage/hooks/query";
import { useAccountEntries } from "@/feature/account-transaction/hooks/query";

import { AccountStateMessage } from "@/components/account/account-state-message";
import { AccountSummaryCard } from "@/components/account/account-summary-card";
import { AccountDetailsCard } from "@/components/account/account-details-card";
import { AccountEntriesCard } from "@/components/account/account-entries-card";

export function AccountDetailView({ accountId }: { accountId: number }) {
  const {
    data: accountEntries,
    isLoading: accountEntriesLoading,
    isError: accountEntriesError,
  } = useAccountEntries(accountId, { page: 1, limit: 25 });

  const {
    data: accountDetail,
    isLoading: accountDetailLoading,
    isError: accountDetailError,
  } = useAccountDetail(accountId);

  const account = accountDetail?.data;

  if (accountEntriesLoading || accountDetailLoading) {
    return (
      <AccountStateMessage
        title="Loading account"
        description="Fetching the account activity..."
      />
    );
  }

  if (accountEntriesError || accountDetailError) {
    return (
      <AccountStateMessage
        title="Unable to load account"
        description="Please try again in a moment."
      />
    );
  }

  if (!account) {
    return (
      <AccountStateMessage
        title="Account not found"
        description="This account may have been removed or is unavailable."
      />
    );
  }

  return (
    <main className="flex flex-1 flex-col bg-zinc-50 px-4 py-6 dark:bg-black sm:px-6">
      <div className="mx-auto w-full max-w-6xl space-y-6">
        <Link
          href="/dashboard"
          className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" aria-hidden />
          Back to dashboard
        </Link>

        <section className="grid gap-4 lg:grid-cols-[1fr_280px]">
          <AccountSummaryCard account={account} />
          <AccountDetailsCard account={account} />
        </section>

        <AccountEntriesCard
          accountName={account.name}
          accountId={accountId}
          currency={account.currency}
          entries={accountEntries?.data ?? []}
        />
      </div>
    </main>
  );
}
