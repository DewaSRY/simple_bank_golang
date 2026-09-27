import { useTranslation } from "react-i18next";
import { AccountWithUserName, useAccounts } from "@/feature/account";
import { SearchInput } from "@/components/common/search-input";

import { AccountCard } from "./account-card";

import { useDepositeStore } from "./store";
import { useState } from "react";
import { DepositeForm } from "./deposite-dialog";
import EmptyState from "@/components/ui/empty-state";

interface Props {
  form: DepositeForm;
}

export function AccountList({ form }: Props) {
  const { t } = useTranslation("common");
  const { setSelectedAccount, selectedAccount } = useDepositeStore();

  const [name, setName] = useState("");

  const { data: accounts } = useAccounts({
    name: name,
    page: 1,
    limit: 10,
  });

  function handleSearch(value: string) {
    setName(value);
  }

  function handleSelectAccount(account: AccountWithUserName) {
    setSelectedAccount(account);
    form.setValue("accountId", account.id, { shouldDirty: true });
  }

  const accountsList = accounts?.data ?? [];
  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* Sticky Search */}
      <div className="sticky top-0 bg-muted z-10 ">
        <SearchInput
          search={name}
          onSearch={handleSearch}
          placeholder={t("searchPlaceholder")}
        />
      </div>

      {/* Scrollable Account List */}
      <div className="min-h-0 flex-1 overflow-y-auto space-y-2 py-2">
        {accountsList.length > 0 ? (
          accountsList.map((account, idx) => (
            <AccountCard
              key={`account-${account.id}-${idx}`}
              name={account.name}
              number={account.number}
              selected={selectedAccount?.id === account.id}
              onClick={() => handleSelectAccount(account)}
            />
          ))
        ) : (
          <EmptyState
            title={t("noAccountsFound")}
            description={t("noAccountsFoundDescription")}
          />
        )}
      </div>
    </div>
  );
}
