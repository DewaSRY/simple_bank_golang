import { useAccounts } from "@/feature/account/hooks/query";
import { SearchInput } from "@/components/common/search-input";

import { AccountCard } from "./account-card";

import { useDepositeStore } from "./store";

export function AccountList() {
  const { setSelectedAccount, selectedAccount } = useDepositeStore();
  const { data: accounts, refetch } = useAccounts();

  function handleSearch(value: string) {
    console.log(value);
    refetch();
  }

  return (
    <div>
      <SearchInput search="" onSearch={handleSearch} />

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
