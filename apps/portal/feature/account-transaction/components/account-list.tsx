import { useState } from "react";
import { useTranslation } from "react-i18next";
import { SearchX, Wallet } from "lucide-react";

import { AccountWithUserName, useAccounts } from "@/feature/account";
import { SearchInput } from "@/components/common/search-input";
import {
  AccountOption,
  AccountOptionSkeleton,
} from "@/components/common/account-option";
import EmptyState from "@/components/ui/empty-state";
import { formatAccountAmount } from "@/lib/number";

import { useDepositeStore } from "./store";
import { DepositeForm } from "./deposite-dialog";

interface Props {
  form: DepositeForm;
}

export function AccountList({ form }: Props) {
  const { t } = useTranslation("common");
  const { setSelectedAccount, selectedAccount } = useDepositeStore();

  const [name, setName] = useState("");

  const { data: accounts, isLoading } = useAccounts({
    name: name,
    page: 1,
    limit: 10,
  });

  function handleSelectAccount(account: AccountWithUserName) {
    setSelectedAccount(account);
    form.setValue("accountId", account.id, { shouldDirty: true });
  }

  const accountsList = accounts?.data ?? [];
  return (
    <div className="flex min-h-0 flex-col gap-3">
      <SearchInput
        search={name}
        onSearch={setName}
        placeholder={t("searchAccountsPlaceholder")}
      />

      <div className="-mx-1 max-h-80 min-h-0 flex-1 space-y-2 overflow-y-auto px-1 py-1">
        {isLoading ? (
          <AccountOptionSkeleton />
        ) : accountsList.length > 0 ? (
          accountsList.map((account, idx) => (
            <AccountOption
              key={`account-${account.id}-${idx}`}
              name={account.name}
              number={account.number}
              balance={formatAccountAmount(account.balance, account.currency)}
              selected={selectedAccount?.id === account.id}
              onClick={() => handleSelectAccount(account)}
            />
          ))
        ) : (
          <EmptyState
            size="sm"
            icon={name ? SearchX : Wallet}
            title={t("noAccountsFound")}
            description={t("noAccountsFoundDescription")}
          />
        )}
      </div>
    </div>
  );
}
