import { notFound } from "next/navigation";
import { isAppLocale } from "@/i18n/settings";
import {
  QueryClient,
  HydrationBoundary,
  dehydrate,
} from "@tanstack/react-query";
import { verifySession } from "@/feature/auth/dal";
import { SessionGuard } from "@/feature/auth";

import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/navigation/app-sidebar";
import { SiteHeader } from "@/components/navigation/site-header";

import { queryKeys, listMeAccountsAction } from "@/feature/account";
import { unwrapActionResult } from "@/lib/api/action-result";

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

  // Matches NavAccountList's own useAccounts({ page: 1, limit: 10, name: "" })
  // call exactly, since TanStack Query hashes the params object into the
  // cache key — a mismatched shape here would prefetch a key the sidebar
  // never reads and it would fall back to fetching client-side anyway.
  const query = {
    page: 1,
    limit: 10,
    name: "",
  };
  await queryClient.prefetchQuery({
    queryKey: queryKeys.list(query),
    queryFn: () => listMeAccountsAction(query).then(unwrapActionResult),
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
      <HydrationBoundary state={dehydrate(queryClient)}>
        <SessionGuard />
        <AppSidebar variant="inset" />
        <SidebarInset>
          <SiteHeader locale={locale} />

          {children}
        </SidebarInset>
      </HydrationBoundary>
    </SidebarProvider>
  );
}
