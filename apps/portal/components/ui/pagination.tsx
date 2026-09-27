"use client";

import React from "react";
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  MoreHorizontal,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/components/ui/button";

interface PaginationProps {
  currentPage: number;
  onPageChange: (page: number) => void;
  rowsPerPage?: number;
  totalRows: number;
  onRowsPerPageChange?: (rows: number) => void;
  rowsPerPageOptions?: number[];
  className?: string;
  pageName?: string; // Optional prop for test ID prefix
}

const Pagination: React.FC<PaginationProps> = ({
  currentPage,
  onPageChange,
  rowsPerPage = 25,
  totalRows,
  onRowsPerPageChange,
  rowsPerPageOptions = [10, 25, 50, 100],
  className,
  pageName,
}) => {
  const { t } = useTranslation("common");
  const totalPages = Math.max(1, Math.ceil(totalRows / rowsPerPage));
  const getPageNumbers = () => {
    const pages: (number | string)[] = [];

    if (totalPages <= 7) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      pages.push(1);

      if (currentPage <= 3) {
        pages.push(2, 3, 4, 5);
        pages.push("ellipsis-end");
        pages.push(totalPages);
      } else if (currentPage >= totalPages - 2) {
        pages.push("ellipsis-start");
        pages.push(
          totalPages - 4,
          totalPages - 3,
          totalPages - 2,
          totalPages - 1,
        );
        pages.push(totalPages);
      } else {
        pages.push("ellipsis-start");
        pages.push(currentPage - 1, currentPage, currentPage + 1);
        pages.push("ellipsis-end");
        pages.push(totalPages);
      }
    }

    return pages;
  };

  const handlePageClick = (page: number) => {
    if (page >= 1 && page <= totalPages && page !== currentPage) {
      onPageChange(page);
    }
  };

  const pageNumbers = getPageNumbers();
  const firstRow = totalRows === 0 ? 0 : (currentPage - 1) * rowsPerPage + 1;
  const lastRow = Math.min(currentPage * rowsPerPage, totalRows);

  return (
    <nav
      aria-label={t("pagination")}
      className={cn(
        "flex w-full flex-col items-center justify-between gap-3 px-4 py-3 sm:flex-row",
        className,
      )}
    >
      <div className="flex shrink-0 items-center gap-3">
        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          <span className="whitespace-nowrap">{t("rowsPerPage")}</span>
          <span className="relative shrink-0">
            <select
              data-testid={`select_${pageName}_rows_per_page`}
              value={rowsPerPage}
              onChange={(e) => onRowsPerPageChange?.(Number(e.target.value))}
              className="h-8 appearance-none rounded-md border border-input bg-card py-1 pr-8 pl-3 text-sm text-foreground shadow-xs transition-[border-color,box-shadow] outline-none hover:border-ring/60 focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
            >
              {rowsPerPageOptions.map((option) => (
                <option
                  key={option}
                  value={option}
                  data-testid={`option_${pageName}_rows_per_page_${option}`}
                >
                  {option}
                </option>
              ))}
            </select>
            <ChevronDown
              className="pointer-events-none absolute top-1/2 right-2 size-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden
            />
          </span>
        </label>
        <span className="text-sm whitespace-nowrap text-muted-foreground tabular-nums">
          {t("paginationRange", {
            from: firstRow,
            to: lastRow,
            count: totalRows,
          })}
        </span>
      </div>

      <div className="flex max-w-full items-center gap-1 overflow-x-auto">
        <button
          type="button"
          onClick={() => handlePageClick(currentPage - 1)}
          disabled={currentPage === 1}
          className={buttonVariants({ variant: "outline", size: "icon-sm" })}
          aria-label={t("previousPage")}
          data-testid={pageName ? `btn_${pageName}_previous_page` : undefined}
        >
          <ChevronLeft aria-hidden />
        </button>

        {pageNumbers.map((page, index) => {
          if (typeof page === "string") {
            return (
              <span
                key={`${page}-${index}`}
                aria-hidden
                className="flex size-8 shrink-0 items-center justify-center text-muted-foreground"
              >
                <MoreHorizontal className="size-4" />
              </span>
            );
          }

          const isCurrent = currentPage === page;
          return (
            <button
              type="button"
              key={page}
              onClick={() => handlePageClick(page)}
              className={cn(
                buttonVariants({
                  variant: isCurrent ? "default" : "ghost",
                  size: "icon-sm",
                }),
                "min-w-8 w-auto px-2 tabular-nums",
                !isCurrent && "text-muted-foreground",
              )}
              aria-label={t("pageNumber", { page })}
              aria-current={isCurrent ? "page" : undefined}
              data-testid={
                pageName ? `btn_${pageName}_page_${page}` : undefined
              }
            >
              {page}
            </button>
          );
        })}

        <button
          type="button"
          onClick={() => handlePageClick(currentPage + 1)}
          disabled={currentPage === totalPages}
          className={buttonVariants({ variant: "outline", size: "icon-sm" })}
          aria-label={t("nextPage")}
          data-testid={pageName ? `btn_${pageName}_next_page` : undefined}
        >
          <ChevronRight aria-hidden />
        </button>
      </div>
    </nav>
  );
};

export default Pagination;
