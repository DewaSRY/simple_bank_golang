"use client";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
} from "@/components/ui/sidebar";
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
      <SidebarGroupLabel>Accounts</SidebarGroupLabel>
      <SidebarGroupContent className="flex flex-col gap-0.5">
        {isLoading ? (
          <div className="flex flex-col gap-3 px-2 py-2">
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
        ) : accounts?.length ? (
          accounts.map((account) => (
            <Link
              key={account.id}
              href={`/account/${account.id}`}
              className="flex items-center justify-between gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            >
              <div className="flex min-w-0 flex-col">
                <span className="truncate font-medium">
                  {account.name || account.username}
                </span>
                <span className="truncate text-xs text-muted-foreground">
                  {account.number}
                </span>
              </div>

              <div className="flex shrink-0 items-baseline gap-1 font-mono text-xs text-muted-foreground">
                <span>{formatBalance(account.balance)}</span>
                <span>{account.currency}</span>
              </div>
            </Link>
          ))
        ) : (
          <p className="px-2 py-2 text-xs text-muted-foreground">
            No accounts yet.
          </p>
        )}
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
