"use client";

import * as React from "react";
import { IconInnerShadowTop } from "@tabler/icons-react";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { NavAccountList } from "./nav-account-list";
import { Separator } from "@base-ui/react";

import { CreateNewAccount } from "../create-account-model/create-new-account";
import { DepositeSideModel } from "@/components/deposite-model/deposite-side-model";
import { TransferSideModel } from "@/components/transfer-modal/transfer-side-model";

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              className="data-[slot=sidebar-menu-button]:px-1.5!"
              render={<a href="#" />}
            >
              <span className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <IconInnerShadowTop className="size-4!" />
              </span>
              <span className="text-base font-semibold tracking-tight">
                Simple Bank
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent className="gap-4">
        <Separator />
        <div className="flex flex-col gap-1 px-2">
          <CreateNewAccount />
          <DepositeSideModel />
          <TransferSideModel />
        </div>

        <Separator />
        <NavAccountList />
      </SidebarContent>
      <SidebarFooter></SidebarFooter>
    </Sidebar>
  );
}
