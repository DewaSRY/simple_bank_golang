import { useTranslation } from "react-i18next";
import { useAccounts } from "@/feature/account/hooks/query";
import { SearchInput } from "@/components/common/search-input";

import { AccountCard } from "./account-card";

import { useDepositeStore } from "./store";
import { useState } from "react";

export function AccountList() {
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

  return (
    <div>
      <SearchInput
        search={name}
        onSearch={handleSearch}
        placeholder={t("searchPlaceholder")}
      />

      <div className="space-y-2 py-2">
        {accounts &&
          accounts.length > 0 &&
          accounts.map((account, idx) => (
            <AccountCard
              key={`account-${account.id}-${idx}`}
              name={account.name}
              number={account.number}
              selected={selectedAccount?.id === account.id}
              onClick={() => setSelectedAccount(account)}
            />
          ))}
      </div>
    </div>
  );
}
