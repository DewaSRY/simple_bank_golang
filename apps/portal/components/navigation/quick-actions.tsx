"use client";

import { ArrowDownToLine, ArrowLeftRight, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { CreateAccountDialog } from "@/feature/account";
import { DepositeDialog } from "@/feature/account-transaction";
import { TransferDialog } from "@/feature/transfer";

import { useQuickActionStore, type QuickAction } from "./quick-action-store";

/** Mounts each app-wide action dialog exactly once, driven by the store. */
export function QuickActionDialogs() {
  const active = useQuickActionStore((s) => s.active);
  const close = useQuickActionStore((s) => s.close);

  const setOpenFor = (kind: QuickAction) => (open: boolean) => {
    if (!open && active === kind) close();
  };

  return (
    <>
      <CreateAccountDialog
        open={active === "create-account"}
        setOpen={setOpenFor("create-account")}
      />
      <DepositeDialog
        open={active === "deposit"}
        setOpen={setOpenFor("deposit")}
      />
      <TransferDialog
        open={active === "transfer"}
        setOpen={setOpenFor("transfer")}
      />
    </>
  );
}

/** Sidebar "Quick actions" group. */
export function SidebarQuickActions() {
  const { t } = useTranslation("common");
  const { t: tTransfer } = useTranslation("transfer");
  const { t: tDeposit } = useTranslation("deposit");
  const { t: tAccount } = useTranslation("account");
  const { isMobile, setOpenMobile } = useSidebar();
  const { openTransfer, openDeposit, openCreateAccount } =
    useQuickActionStore();

  // On mobile the sidebar is a sheet — close it so the dialog isn't stacked
  // on top of an open drawer.
  const run = (action: () => void) => () => {
    if (isMobile) setOpenMobile(false);
    action();
  };

  const items = [
    {
      label: tTransfer("transferFundsTitle"),
      Icon: ArrowLeftRight,
      onClick: run(() => openTransfer()),
    },
    {
      label: tDeposit("depositFundsTitle"),
      Icon: ArrowDownToLine,
      onClick: run(() => openDeposit()),
    },
    {
      label: tAccount("createNewAccount"),
      Icon: Plus,
      onClick: run(openCreateAccount),
    },
  ];

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("quickActions")}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map(({ label, Icon, onClick }) => (
            <SidebarMenuItem key={label}>
              <SidebarMenuButton onClick={onClick}>
                <Icon aria-hidden />
                <span>{label}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

/** Page-level action buttons for the dashboard header. */
export function DashboardQuickActions() {
  const { t: tTransfer } = useTranslation("transfer");
  const { t: tDeposit } = useTranslation("deposit");
  const { t: tAccount } = useTranslation("account");
  const { openTransfer, openDeposit, openCreateAccount } =
    useQuickActionStore();

  return (
    <>
      <Button variant="outline" onClick={openCreateAccount}>
        <Plus aria-hidden />
        {tAccount("newAccount")}
      </Button>
      <Button variant="outline" onClick={() => openDeposit()}>
        <ArrowDownToLine aria-hidden />
        {tDeposit("depositAction")}
      </Button>
      <Button onClick={() => openTransfer()}>
        <ArrowLeftRight aria-hidden />
        {tTransfer("transferAction")}
      </Button>
    </>
  );
}
