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
import { TransferSideModel } from "@/components/transfer-modal/transfer-side-model";
export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton className="data-[slot=sidebar-menu-button]:px-1.5!">
              <a href="#" className="flex items-center gap-2">
                <IconInnerShadowTop className="size-5!" />
                <span className="text-base font-semibold">SIMPLE BANK</span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <CreateNewAccount />
        <TransferSideModel />
        <Separator className="bg-gray-100 h-1 rounded-sm" />
        <NavAccountList />
      </SidebarContent>
      <SidebarFooter></SidebarFooter>
    </Sidebar>
  );
}
