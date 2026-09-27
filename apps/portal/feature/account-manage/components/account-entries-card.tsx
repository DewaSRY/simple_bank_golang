"use client";

import { useMemo, useTransition } from "react";
import { useTranslation } from "react-i18next";
import { useTable } from "@tanstack/react-table";
import { ReceiptText } from "lucide-react";
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
import EmptyState from "@/components/ui/empty-state";
import {
  accountEntriesTableFeatures,
  getAccountEntriesColumns,
} from "./account-entries-columns";
import {
  useAccountEntries,
  type DisplayLedgerEntry,
} from "@/feature/account-transaction";
import { useQueryStates, parseAsInteger } from "nuqs";
import { cn } from "@/lib/utils";

import Pagination from "@/components/ui/pagination";

const EMPTY_ENTRIES: DisplayLedgerEntry[] = [];

type ColumnMeta = { align?: "left" | "right"; hideOnMobile?: boolean };

function cellClassName(meta: ColumnMeta | undefined, base: string) {
  return cn(
    base,
    meta?.align === "right" && "text-right",
    meta?.hideOnMobile && "hidden sm:table-cell",
  );
}

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
  const [isPending, startTransition] = useTransition();
  const queryStateOptions = { shallow: false as const, startTransition };

  const [{ page, limit }, setQuery] = useQueryStates({
    page: parseAsInteger.withOptions(queryStateOptions).withDefault(1),
    limit: parseAsInteger.withOptions(queryStateOptions).withDefault(25),
  });

  const { data: accountEntries, isFetching } = useAccountEntries(accountId, {
    page,
    limit,
  });

  const columns = useMemo(
    () => getAccountEntriesColumns({ currency, t }),
    [currency, t],
  );

  const table = useTable({
    features: accountEntriesTableFeatures,
    columns,
    data: accountEntries?.data ?? EMPTY_ENTRIES,
  });

  const isUpdating = isPending || (isFetching && !!accountEntries);

  return (
    <Card className="gap-0 pb-0">
      <CardHeader className="border-b">
        <CardTitle>{t("accountEntriesTitle")}</CardTitle>
        <CardDescription>
          {t("accountEntriesDescription", { name: accountName })}
        </CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        {accountEntries?.data.length === 0 ? (
          <EmptyState
            icon={ReceiptText}
            title={t("noEntriesTitle")}
            description={t("noEntriesDescription")}
          />
        ) : (
          <div
            aria-busy={isUpdating}
            className={cn(
              "transition-opacity duration-200",
              isUpdating && "pointer-events-none opacity-60",
            )}
          >
            <Table className="sm:min-w-160">
              <TableHeader className="bg-muted/40 text-xs tracking-wide text-muted-foreground uppercase">
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className="hover:bg-transparent">
                    {headerGroup.headers.map((header) => (
                      <TableHead
                        key={header.id}
                        className={cellClassName(
                          header.column.columnDef.meta,
                          "h-10 px-4 font-medium sm:px-6",
                        )}
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
                  <TableRow
                    key={row.id}
                    className="transition-colors hover:bg-muted/40"
                  >
                    {row.getAllCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className={cellClassName(
                          cell.column.columnDef.meta,
                          "px-4 py-3.5 align-middle sm:px-6",
                        )}
                      >
                        <table.FlexRender cell={cell} />
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              className="border-t"
              currentPage={accountEntries?.meta?.page || 1}
              onPageChange={(p) =>
                void setQuery((prev) => {
                  prev.page = p;
                  return prev;
                })
              }
              totalRows={accountEntries?.meta?.total || 0}
              rowsPerPageOptions={[25, 50, 100]}
              rowsPerPage={accountEntries?.meta?.limit || 25}
              onRowsPerPageChange={(l) =>
                void setQuery((prev) => {
                  prev.limit = l;
                  // A new page size can make the current page out of range.
                  prev.page = 1;
                  return prev;
                })
              }
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
