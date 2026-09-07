import { notFound } from "next/navigation";
import { isAppLocale } from "@/i18n/settings";
import { QueryClient } from "@tanstack/react-query";
import { verifySession } from "@/feature/auth/dal";

import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/navigation/app-sidebar";
import { SiteHeader } from "@/components/navigation/site-header";

import { queryKeys } from "@/feature/account/hooks/query";
import { accountClient } from "@/feature/account/client";

export default async function ProtectedLayout({
  children,
  params,
}: LayoutProps<"/[locale]">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  await verifySession(locale);

  const queryClient = new QueryClient();

  const query = {
    page: 1,
    limit: 10,
  };
  await queryClient.prefetchQuery({
    queryKey: queryKeys.list(query),
    queryFn: () => accountClient.listAccounts(query).then((res) => res.data),
  });

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 72)",
          "--header-height": "calc(var(--spacing) * 12)",
        } as React.CSSProperties
      }
    >
      <AppSidebar variant="inset" />
      <SidebarInset>
        <SiteHeader locale={locale} />

        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
