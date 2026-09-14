"use client";
import { SidebarGroup, SidebarGroupContent } from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";

import { formatBalance } from "@/lib/number";
import { useAccounts } from "@/feature/account/hooks/query";

import Link from "next/link";

export function NavAccountList() {
  const { data: accounts, isLoading } = useAccounts({
    page: 1,
    limit: 10,
    name: "",
  });

  return (
    <SidebarGroup>
      <SidebarGroupContent className="flex flex-col gap-2">
        <div className="mb-2 flex justify-between">
          <h2 className="text-lg font-bold">Accounts</h2>
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-3 py-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between">
                <div className="flex flex-col gap-1.5">
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="h-3 w-16" />
                </div>
                <Skeleton className="h-3 w-10" />
              </div>
            ))}
          </div>
        ) : (
          <div>
            {accounts?.map((account) => (
              <Link
                key={account.id}
                className="min-h-min! py-2 "
                role="button"
                href={`/account/${account.id}`}
              >
                <div className="flex flex-col justify-between ">
                  <h2 className="text-sm ">
                    {account.name || account.username}
                  </h2>
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
              </Link>
            ))}
          </div>
        )}
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
