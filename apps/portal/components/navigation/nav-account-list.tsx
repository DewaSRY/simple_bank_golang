"use client";

import { SidebarGroup, SidebarGroupContent } from "@/components/ui/sidebar";

import { formatBalance } from "@/lib/number";
import { useAccounts } from "@/feature/account/hooks/query";

export function NavAccountList() {
  const { data: accounts, isLoading } = useAccounts();

  return (
    <SidebarGroup>
      <SidebarGroupContent className="flex flex-col gap-2">
        <div>
          <h2 className="text-lg font-bold">Accounts</h2>
        </div>

        <div>
          {accounts?.map((account) => (
            <div key={account.id} className="min-h-min! py-2 ">
              <div className="flex flex-col justify-between ">
                <h2 className="text-sm ">{account.name || account.username}</h2>
                <div>
                  <span className="text-right text-xs text-gray-500">
                    {account.number}
                  </span>
                </div>
              </div>

              <div className="flex justify-end text-xs  ">
                <span>{formatBalance(account.balance)}</span>
                <span className="ml-0.5">{account.currency}</span>
              </div>
            </div>
          ))}
        </div>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
