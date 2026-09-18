"use client";

import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useTable } from "@tanstack/react-table";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  accountEntriesTableFeatures,
  getAccountEntriesColumns,
} from "@/components/account/account-entries-columns";
import { useAccountEntries } from "@/feature/account-transaction/hooks/query";
import type { AccountEntriesResponse } from "@/feature/account-transaction/type";

const EMPTY_ENTRIES: AccountEntriesResponse[] = [];

export function AccountEntriesCard({
  accountName,
  accountId,
  currency,
}: {
  accountName: string;
  accountId: number;
  currency: string;
}) {
  const { t } = useTranslation("account");

  const {
    data: accountEntries,
    isLoading: accountEntriesLoading,
    isError: accountEntriesError,
  } = useAccountEntries(accountId, { page: 1, limit: 25 });

  const columns = useMemo(
    () => getAccountEntriesColumns({ accountId, currency, t }),
    [accountId, currency, t],
  );

  const table = useTable({
    features: accountEntriesTableFeatures,
    columns,
    data: accountEntries?.data ?? EMPTY_ENTRIES,
  });

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("accountEntriesTitle")}</CardTitle>
        <CardDescription>
          {t("accountEntriesDescription", { name: accountName })}
        </CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        {accountEntries?.data.length === 0 ? (
          <div className="px-6 py-14 text-center">
            <p className="font-medium">{t("noEntriesTitle")}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("noEntriesDescription")}
            </p>
          </div>
        ) : (
          <Table className="min-w-175">
            <TableHeader className="bg-muted/40 text-xs uppercase tracking-wide text-muted-foreground">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={
                        header.column.columnDef.meta?.align === "right"
                          ? "px-4 py-3 text-right"
                          : "px-4 py-3"
                      }
                    >
                      {header.isPlaceholder ? null : (
                        <table.FlexRender header={header} />
                      )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} className="hover:bg-muted/30">
                  {row.getAllCells().map((cell) => (
                    <TableCell
                      key={cell.id}
                      className={
                        cell.column.columnDef.meta?.align === "right"
                          ? "px-4 py-4 text-right"
                          : "px-4 py-4"
                      }
                    >
                      <table.FlexRender cell={cell} />
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
