import type { AppLocale } from "@/i18n/settings";

import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { LogoutButton } from "@/feature/auth/components/logout-button";

import { LocaleSwitcher } from "@/components/locale-switcher";
import { ThemeToggle } from "@/components/theme-toggle";

export function SiteHeader({ locale }: { locale: AppLocale }) {
  return (
    <header className="sticky top-0 z-20 flex h-(--header-height) shrink-0 items-center border-b bg-background/80 backdrop-blur-md transition-[width,height] ease-linear md:rounded-t-xl">
      <div className="flex w-full items-center gap-2 px-4 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <div className="ml-auto flex items-center gap-1.5">
          <LocaleSwitcher />
          <ThemeToggle />
          <Separator
            orientation="vertical"
            className="mx-1.5 data-[orientation=vertical]:h-5"
          />
          <LogoutButton locale={locale} />
        </div>
      </div>
    </header>
  );
}
