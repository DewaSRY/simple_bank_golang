"use client";

import { useState } from "react";
import {
  ArrowDownToLine,
  ArrowLeftRight,
  MoreHorizontal,
  Pencil,
  Trash2,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useRouter } from "@/i18n/navigation";

import { useAccountDetail } from "@/feature/account-manage";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { PageContainer, PageHeader } from "@/components/common/page-header";
import { useQuickActionStore } from "@/components/navigation/quick-action-store";

import { AccountStateMessage } from "./account-state-message";
import { AccountSummaryCard } from "./account-summary-card";
import { AccountDetailsCard } from "./account-details-card";
import { AccountEntriesCard } from "./account-entries-card";
import { EditAccountDialog } from "./edit-account-dialog";
import { DeleteAccountDialog } from "@/feature/account-manage/components/delete-account-dialog";

export function AccountDetailView({ accountId }: { accountId: number }) {
  const { t } = useTranslation("account");
  const { t: tCommon } = useTranslation("common");
  const { t: tTransfer } = useTranslation("transfer");
  const { t: tDeposit } = useTranslation("deposit");
  const router = useRouter();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const { openDeposit, openTransfer } = useQuickActionStore();

  const { data: accountDetail } = useAccountDetail(accountId);

  const account = accountDetail?.data;

  if (!account) {
    return (
      <AccountStateMessage
        title={t("notFoundTitle")}
        description={t("notFoundDescription")}
      />
    );
  }

  const preselected = {
    id: account.id,
    name: account.name,
    number: account.number,
    currency: account.currency,
  };

  return (
    <PageContainer>
      <PageHeader
        breadcrumbLabel={tCommon("breadcrumb")}
        breadcrumbs={[
          <Link
            key="dashboard"
            href="/dashboard"
            className="rounded-sm transition-colors outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
          >
            {tCommon("dashboardTitle")}
          </Link>,
        ]}
        title={account.name}
        titleAddon={
          account.is_main ? (
            <Badge variant="warning">{t("mainAccount")}</Badge>
          ) : null
        }
        actions={
          <>
            <Button variant="outline" onClick={() => openDeposit(preselected)}>
              <ArrowDownToLine aria-hidden />
              {tDeposit("depositAction")}
            </Button>
            <Button onClick={() => openTransfer(preselected)}>
              <ArrowLeftRight aria-hidden />
              {tTransfer("transferAction")}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="outline"
                    size="icon"
                    aria-label={tCommon("moreActions")}
                  />
                }
              >
                <MoreHorizontal aria-hidden />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-44">
                <DropdownMenuItem onClick={() => setEditOpen(true)}>
                  <Pencil aria-hidden />
                  {t("edit")}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => setDeleteOpen(true)}
                >
                  <Trash2 aria-hidden />
                  {t("delete")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </>
        }
      />

      <EditAccountDialog
        account={account}
        open={editOpen}
        setOpen={setEditOpen}
      />
      <DeleteAccountDialog
        account={account}
        open={deleteOpen}
        setOpen={setDeleteOpen}
        onDeleted={() => router.push("/dashboard")}
      />

      <section className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <AccountSummaryCard account={account} />
        <AccountDetailsCard account={account} />
      </section>

      <AccountEntriesCard
        accountName={account.name}
        accountId={accountId}
        currency={account.currency}
      />
    </PageContainer>
  );
}
