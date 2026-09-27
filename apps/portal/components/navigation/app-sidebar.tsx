"use client";

import * as React from "react";
import { IconWallet } from "@tabler/icons-react";
import { LayoutDashboard } from "lucide-react";
import { useTranslation } from "react-i18next";
import { siGithub } from "simple-icons";

import { Link, usePathname } from "@/i18n/navigation";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { NavAccountList } from "./nav-account-list";
import { SidebarQuickActions } from "./quick-actions";

const SOURCE_URL = "https://github.com/DewaSRY/simple_bank_golang";

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { t } = useTranslation("common");
  const pathname = usePathname();

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              render={<Link href="/dashboard" />}
              aria-label={t("appName")}
            >
              <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-xs">
                <IconWallet className="size-4!" aria-hidden />
              </span>
              <span className="text-base font-semibold tracking-tight">
                {t("appName")}
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent className="gap-0">
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  isActive={pathname === "/dashboard"}
                  render={<Link href="/dashboard" />}
                >
                  <LayoutDashboard aria-hidden />
                  <span>{t("dashboardTitle")}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarQuickActions />
        <SidebarSeparator className="my-1" />
        <NavAccountList />
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="sm"
              className="text-sidebar-foreground/70"
              render={
                <a
                  href={SOURCE_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                />
              }
            >
              <svg
                role="img"
                aria-hidden
                viewBox="0 0 24 24"
                className="size-3.5"
                fill="currentColor"
              >
                <path d={siGithub.path} />
              </svg>
              <span>{t("viewSource")}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
