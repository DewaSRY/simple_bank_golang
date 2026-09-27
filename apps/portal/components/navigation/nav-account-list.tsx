"use client";
import { useTranslation } from "react-i18next";
import { SearchX, Star, Wallet } from "lucide-react";
import { useQueryStates, parseAsString, parseAsInteger } from "nuqs";

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  useSidebar,
} from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import EmptyState from "@/components/ui/empty-state";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { SearchInput } from "../common/search-input";

import { cn } from "@/lib/utils";
import { formatBalance } from "@/lib/number";
import { Link, usePathname } from "@/i18n/navigation";
import { useAccounts } from "@/feature/account";

export function NavAccountList() {
  const { t } = useTranslation("common");
  const pathname = usePathname();
  const { isMobile, setOpenMobile } = useSidebar();

  // Shallow: no Server Component reads these params, so a server round
  // trip per keystroke would only re-render the current page for nothing.
  const [{ account_page, account_limit, search_account }, setQuery] =
    useQueryStates({
      account_page: parseAsInteger.withDefault(1),
      account_limit: parseAsInteger.withDefault(25),
      search_account: parseAsString.withDefault(""),
    });

  const { data: accountsResponse, isLoading } = useAccounts({
    page: account_page,
    limit: account_limit,
    name: search_account,
  });

  const accounts = accountsResponse?.data ?? [];

  function handleSearch(value: string) {
    setQuery({ search_account: value || null });
  }

  return (
    <SidebarGroup className="flex min-h-0 flex-1 flex-col">
      <SidebarGroupLabel className="justify-between">
        <span>{t("yourAccounts")}</span>
        {accountsResponse?.meta?.total ? (
          <span className="tabular-nums">{accountsResponse.meta.total}</span>
        ) : null}
      </SidebarGroupLabel>
      <SidebarGroupContent className="flex min-h-0 min-w-0 flex-1 flex-col gap-1">
        <div className="px-0.5 pb-1">
          <SearchInput
            className="h-9"
            search={search_account}
            onSearch={handleSearch}
            placeholder={t("searchAccountsPlaceholder")}
          />
        </div>

        <div className="-mx-1 flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto px-1">
          {isLoading ? (
            <div className="flex flex-col gap-3 px-2.5 py-2" aria-hidden>
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex flex-col gap-1.5">
                  <Skeleton className="h-3.5 w-28" />
                  <Skeleton className="h-3 w-20" />
                </div>
              ))}
            </div>
          ) : accounts.length > 0 ? (
            accounts.map((account) => {
              const href = `/account/${account.id}`;
              const isActive = pathname === href;

              return (
                <Link
                  key={account.id}
                  href={href}
                  aria-current={isActive ? "page" : undefined}
                  onClick={() => isMobile && setOpenMobile(false)}
                  className={cn(
                    "group/account relative flex flex-col gap-0.5 rounded-md px-2.5 py-2 text-sm outline-none transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring",
                    isActive &&
                      "bg-sidebar-accent font-medium text-sidebar-accent-foreground before:absolute before:inset-y-2 before:left-0 before:w-0.5 before:rounded-full before:bg-sidebar-primary",
                  )}
                >
                  <span className="flex min-w-0 items-center gap-1.5">
                    <span className="truncate font-medium">
                      {account.name || account.username}
                    </span>
                    {account.is_main && (
                      <Tooltip>
                        <TooltipTrigger
                          render={<span className="inline-flex shrink-0" />}
                        >
                          <Star
                            className="size-3 fill-warning text-warning"
                            aria-label={t("mainAccount")}
                          />
                        </TooltipTrigger>
                        <TooltipContent>{t("mainAccount")}</TooltipContent>
                      </Tooltip>
                    )}
                  </span>
                  <span className="flex min-w-0 items-baseline justify-between gap-2 text-xs text-muted-foreground">
                    <span className="truncate">{account.number}</span>
                    <span className="shrink-0 font-mono tabular-nums">
                      {formatBalance(account.balance)} {account.currency}
                    </span>
                  </span>
                </Link>
              );
            })
          ) : (
            <EmptyState
              size="sm"
              icon={search_account ? SearchX : Wallet}
              title={search_account ? t("noAccountsFound") : t("noAccounts")}
            />
          )}
        </div>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
