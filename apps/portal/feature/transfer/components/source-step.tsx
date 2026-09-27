import { useState } from "react";
import { useTranslation } from "react-i18next";
import { SearchX, Wallet } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { SearchInput } from "@/components/common/search-input";
import {
  AccountOption,
  AccountOptionSkeleton,
} from "@/components/common/account-option";
import EmptyState from "@/components/ui/empty-state";
import { useAccounts } from "@/feature/account";
import { formatAccountAmount } from "@/lib/number";

import { useTransferStore } from "./store";
import type { SourceAccount } from "./type";

export function SourceStep() {
  const { t } = useTranslation("transfer");
  const { t: tCommon } = useTranslation("common");
  const { sourceAccount, setSourceAccount, setStep } = useTransferStore();
  const [name, setName] = useState("");

  const { data: accounts, isLoading } = useAccounts({
    name,
    page: 1,
    limit: 10,
  });

  function handleSelect(account: SourceAccount) {
    setSourceAccount(account);
    setStep("destination");
  }

  const accountsList = accounts?.data ?? [];
  return (
    <div className="flex min-h-0 flex-col gap-3">
      <SearchInput
        search={name}
        onSearch={setName}
        placeholder={t("searchAccountsPlaceholder")}
      />

      <div className="-mx-1 max-h-80 space-y-2 overflow-y-auto px-1 py-1">
        {isLoading ? (
          <AccountOptionSkeleton />
        ) : accountsList.length > 0 ? (
          accountsList.map((account, idx) => (
            <AccountOption
              key={`source-${account.id}-${idx}`}
              name={account.name}
              number={account.number}
              balance={formatAccountAmount(account.balance, account.currency)}
              selected={sourceAccount?.id === account.id}
              onClick={() => handleSelect(account)}
            />
          ))
        ) : (
          <EmptyState
            size="sm"
            icon={name ? SearchX : Wallet}
            title={tCommon("noAccountsFound")}
            description={tCommon("noAccountsFoundDescription")}
          />
        )}
      </div>

      <DialogFooter>
        <DialogClose
          render={<Button variant="outline">{tCommon("cancel")}</Button>}
        />
      </DialogFooter>
    </div>
  );
}
