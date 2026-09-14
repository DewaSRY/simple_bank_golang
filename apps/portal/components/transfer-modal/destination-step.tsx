import { useState } from "react";

import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { SearchInput } from "@/components/common/search-input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSearchAccountByNumber } from "@/feature/account/hooks/query";
import { useRecentTransactions } from "@/feature/account-transaction/hooks/query";

import { AccountCard } from "./account-card";
import { useTransferStore } from "./store";
import type { DestinationAccount } from "./type";

export function DestinationStep() {
  const { sourceAccount, destinationAccount, setDestinationAccount, setStep } =
    useTransferStore();
  const [number, setNumber] = useState("");

  const { data: recent } = useRecentTransactions(sourceAccount?.id ?? 0, {
    page: 1,
    limit: 10,
  });

  const { data: searchResults } = useSearchAccountByNumber(
    { number, page: 1, limit: 10 },
    { enabled: number.trim().length > 0 },
  );

  function handleSelect(account: DestinationAccount) {
    setDestinationAccount(account);
    setStep("details");
  }

  const recentResults = (recent?.data ?? []).filter(
    (account) => account.id !== sourceAccount?.id,
  );
  const numberResults = (searchResults ?? []).filter(
    (account) => account.id !== sourceAccount?.id,
  );

  return (
    <div>
      <Tabs defaultValue="recent">
        <TabsList>
          <TabsTrigger value="recent">Recent</TabsTrigger>
          <TabsTrigger value="search">Search by number</TabsTrigger>
        </TabsList>

        <TabsContent value="recent">
          <div className="max-h-80 space-y-2 overflow-y-auto py-2">
            {recentResults.length > 0 ? (
              recentResults.map((account, idx) => (
                <AccountCard
                  key={`recent-${account.id}-${idx}`}
                  name={account.name}
                  number={account.number}
                  subtitle={account.username}
                  selected={destinationAccount?.id === account.id}
                  onClick={() => handleSelect(account)}
                />
              ))
            ) : (
              <p className="py-4 text-center text-sm text-muted-foreground">
                No recent transactions yet.
              </p>
            )}
          </div>
        </TabsContent>

        <TabsContent value="search">
          <SearchInput
            search={number}
            onSearch={setNumber}
            placeholder="Search by account number..."
          />
          <div className="max-h-80 space-y-2 overflow-y-auto py-2">
            {number.trim().length === 0 ? (
              <p className="py-4 text-center text-sm text-muted-foreground">
                Enter an account number to search.
              </p>
            ) : numberResults.length > 0 ? (
              numberResults.map((account, idx) => (
                <AccountCard
                  key={`search-${account.id}-${idx}`}
                  name={account.name}
                  number={account.number}
                  subtitle={account.username}
                  selected={destinationAccount?.id === account.id}
                  onClick={() => handleSelect(account)}
                />
              ))
            ) : (
              <p className="py-4 text-center text-sm text-muted-foreground">
                No accounts found.
              </p>
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
          Back
        </Button>
      </DialogFooter>
    </div>
  );
}
