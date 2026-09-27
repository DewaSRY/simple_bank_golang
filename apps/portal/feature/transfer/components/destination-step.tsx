import { useState } from "react";
import { useTranslation } from "react-i18next";
import { History, SearchX, UserSearch } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { SearchInput } from "@/components/common/search-input";
import {
  AccountOption,
  AccountOptionSkeleton,
} from "@/components/common/account-option";
import EmptyState from "@/components/ui/empty-state";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSearchAccountByNumber } from "@/feature/account";
import { useRecentTransactions } from "@/feature/account-transaction";

import { useTransferStore } from "./store";
import type { DestinationAccount } from "./type";

export function DestinationStep() {
  const { t } = useTranslation("transfer");
  const { t: tCommon } = useTranslation("common");
  const { sourceAccount, destinationAccount, setDestinationAccount, setStep } =
    useTransferStore();
  const [number, setNumber] = useState("");

  const { data: recent, isLoading: isRecentPending } = useRecentTransactions(
    sourceAccount?.id ?? 0,
    {
      page: 1,
      limit: 10,
    },
  );

  const hasQuery = number.trim().length > 0;
  const { data: searchResults, isFetching: isSearching } =
    useSearchAccountByNumber(
      { number, page: 1, limit: 10 },
      { enabled: hasQuery },
    );

  function handleSelect(account: DestinationAccount) {
    setDestinationAccount(account);
    setStep("details");
  }

  const recentResults = (recent?.data ?? []).filter(
    (account) => account.id !== sourceAccount?.id,
  );

  const numberResults = (searchResults?.data ?? []).filter(
    (account) => account.id !== sourceAccount?.id,
  );

  return (
    <div className="flex flex-col gap-3">
      {sourceAccount ? (
        <p className="text-sm text-muted-foreground">
          {t("from")}:{" "}
          <span className="font-medium text-foreground">
            {sourceAccount.name}
          </span>{" "}
          · {sourceAccount.number}
        </p>
      ) : null}

      <Tabs defaultValue="recent">
        <TabsList className="w-full">
          <TabsTrigger value="recent">
            <History aria-hidden />
            {t("recentTab")}
          </TabsTrigger>
          <TabsTrigger value="search">
            <UserSearch aria-hidden />
            {t("searchByNumberTab")}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="recent">
          <div className="-mx-1 max-h-80 space-y-2 overflow-y-auto px-1 py-2">
            {isRecentPending ? (
              <AccountOptionSkeleton />
            ) : recentResults.length > 0 ? (
              recentResults.map((account, idx) => (
                <AccountOption
                  key={`recent-${account.id}-${idx}`}
                  name={account.name}
                  number={account.number}
                  subtitle={account.username}
                  selected={destinationAccount?.id === account.id}
                  onClick={() => handleSelect(account)}
                />
              ))
            ) : (
              <EmptyState
                size="sm"
                icon={History}
                title={t("noRecentTransactions")}
                description={t("noRecentTransactionsHint")}
              />
            )}
          </div>
        </TabsContent>

        <TabsContent value="search" className="space-y-2 pt-1">
          <SearchInput
            search={number}
            onSearch={setNumber}
            placeholder={t("searchByNumberPlaceholder")}
            autoFocus
          />
          <div className="-mx-1 max-h-72 space-y-2 overflow-y-auto px-1 py-1">
            {!hasQuery ? (
              <EmptyState
                size="sm"
                icon={UserSearch}
                title={t("enterAccountNumberPrompt")}
              />
            ) : isSearching && numberResults.length === 0 ? (
              <AccountOptionSkeleton count={2} />
            ) : numberResults.length > 0 ? (
              numberResults.map((account, idx) => (
                <AccountOption
                  key={`search-${account.id}-${idx}`}
                  name={account.name}
                  number={account.number}
                  subtitle={account.username}
                  selected={destinationAccount?.id === account.id}
                  onClick={() => handleSelect(account)}
                />
              ))
            ) : (
              <EmptyState
                size="sm"
                icon={SearchX}
                title={t("noAccountsFound")}
              />
            )}
          </div>
        </TabsContent>
      </Tabs>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={() => setStep("source")}
        >
          {tCommon("back")}
        </Button>
      </DialogFooter>
    </div>
  );
}
