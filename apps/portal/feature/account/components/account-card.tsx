"use client";

import { ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "@/i18n/navigation";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatBalance } from "@/lib/number";
import type { AccountWithUserName } from "@/feature/account";

function initials(name: string) {
  return name.trim().slice(0, 2).toUpperCase() || "?";
}

export function AccountListItem({ account }: { account: AccountWithUserName }) {
  const { t } = useTranslation("common");

  return (
    <Link
      href={`/account/${account.id}`}
      className="group/account-card block rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
    >
      <Card className="h-full gap-4 px-5 py-5! transition-shadow duration-200 group-hover/account-card:shadow-md group-hover/account-card:ring-primary/25">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span
              className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-sm font-semibold text-primary"
              aria-hidden
            >
              {initials(account.name || account.username)}
            </span>
            <div className="flex min-w-0 flex-col">
              <span className="truncate font-medium">{account.name}</span>
              <span className="truncate text-xs text-muted-foreground">
                {account.number}
              </span>
            </div>
          </div>
          {account.is_main ? (
            <Badge variant="warning">{t("mainAccount")}</Badge>
          ) : (
            <ChevronRight
              className="size-4 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover/account-card:opacity-100"
              aria-hidden
            />
          )}
        </div>

        <div className="mt-auto space-y-0.5 border-t pt-4">
          <p className="text-xs text-muted-foreground">
            {t("availableBalance")}
          </p>
          <p className="flex items-baseline gap-1.5">
            <span className="font-mono text-xl font-semibold tracking-tight tabular-nums">
              {formatBalance(account.balance)}
            </span>
            <span className="text-xs font-medium text-muted-foreground">
              {account.currency}
            </span>
          </p>
        </div>
      </Card>
    </Link>
  );
}
