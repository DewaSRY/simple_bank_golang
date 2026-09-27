import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { SearchInput } from "@/components/common/search-input";
import { useAccounts } from "@/feature/account";

import { AccountCard } from "./account-card";
import { useTransferStore } from "./store";
import type { SourceAccount } from "./type";
import EmptyState from "@/components/ui/empty-state";

export function SourceStep() {
  const { t } = useTranslation("transfer");
  const { t: tCommon } = useTranslation("common");
  const { sourceAccount, setSourceAccount, setStep } = useTransferStore();
  const [name, setName] = useState("");

  const { data: accounts } = useAccounts({ name, page: 1, limit: 10 });

  function handleSelect(account: SourceAccount) {
    setSourceAccount(account);
    setStep("destination");
  }

  const accountsList = accounts?.data ?? [];
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="sticky top-0 bg-muted z-10 ">
        <SearchInput
          search={name}
          onSearch={setName}
          placeholder={t("searchAccountsPlaceholder")}
        />
      </div>

      <div className="max-h-80 space-y-2 overflow-y-auto py-2">
        {accountsList.length > 0 ? (
          accountsList.map((account, idx) => (
            <AccountCard
              key={`source-${account.id}-${idx}`}
              name={account.name}
              number={account.number}
              selected={sourceAccount?.id === account.id}
              onClick={() => handleSelect(account)}
            />
          ))
        ) : (
          <EmptyState
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
