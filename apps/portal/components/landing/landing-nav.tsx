"use client";

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Popover } from "@base-ui/react/popover";
import {
  ChevronDown,
  Menu,
  Wallet,
  Layers,
  Server,
  Database,
  ShieldCheck,
} from "lucide-react";
import { motion } from "motion/react";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetClose,
} from "@/components/ui/sheet";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { ThemeToggle } from "@/components/theme-toggle";
import { cn } from "@/lib/utils";

export function LandingNav() {
  const { t } = useTranslation("common");
  const { t: tLanding } = useTranslation("landing");
  const { t: tAuth } = useTranslation("auth");
  const [mobileOpen, setMobileOpen] = useState(false);

  const exploreItems = [
    { icon: Layers, title: tLanding("nav.links.architecture"), href: "#journey" },
    { icon: Server, title: tLanding("nav.links.backend"), href: "#backend" },
    { icon: Database, title: tLanding("nav.links.database"), href: "#database" },
    { icon: ShieldCheck, title: tLanding("nav.links.security"), href: "#security" },
  ];

  return (
    <header className="sticky top-0 z-50 w-full border-b border-transparent bg-background/70 backdrop-blur-md transition-colors">
      <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-4 sm:px-6">
        <Link
          href="/"
          className="flex items-center gap-2 text-base font-semibold tracking-tight"
        >
          <span className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Wallet className="size-4" aria-hidden />
          </span>
          {t("appName")}
        </Link>

        {/* desktop nav */}
        <nav className="hidden items-center gap-1 md:flex">
          <Popover.Root>
            <Popover.Trigger
              render={
                <Button
                  variant="ghost"
                  className="gap-1 text-sm text-muted-foreground hover:text-foreground"
                />
              }
            >
              {tLanding("nav.explore")}
              <ChevronDown className="size-3.5 transition-transform duration-150 group-data-popup-open/button:rotate-180" />
            </Popover.Trigger>
            <Popover.Portal>
              <Popover.Positioner
                className="isolate z-50 outline-none"
                side="bottom"
                align="start"
                sideOffset={12}
              >
                <Popover.Popup className="w-screen max-w-md origin-(--transform-origin) rounded-2xl bg-popover p-2 text-popover-foreground shadow-lg ring-1 ring-foreground/10 duration-150 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95">
                  <p className="px-3 pt-2 pb-1 text-xs text-muted-foreground">
                    {tLanding("nav.exploreDescription")}
                  </p>
                  <div className="flex flex-col gap-1 p-1">
                    {exploreItems.map((item) => (
                      <Popover.Close
                        key={item.title}
                        nativeButton={false}
                        render={
                          <a
                            href={item.href}
                            className="flex items-center gap-3 rounded-xl p-2.5 transition-colors hover:bg-muted"
                          />
                        }
                      >
                        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <item.icon className="size-4.5" aria-hidden />
                        </span>
                        <span className="text-sm font-medium">
                          {item.title}
                        </span>
                      </Popover.Close>
                    ))}
                  </div>
                </Popover.Popup>
              </Popover.Positioner>
            </Popover.Portal>
          </Popover.Root>
        </nav>

        <div className="hidden items-center gap-2 md:flex">
          <LocaleSwitcher />
          <ThemeToggle />
          <Link
            href="/onboarding"
            className="px-3 text-sm font-medium text-muted-foreground hover:text-foreground"
          >
            {tLanding("nav.tryDemo")}
          </Link>
          <Link
            href="/login"
            className="px-3 text-sm font-medium text-muted-foreground hover:text-foreground"
          >
            {tAuth("login")}
          </Link>
          <Button size="sm" nativeButton={false} render={<Link href="/register" />}>
            {tLanding("nav.getStarted")}
          </Button>
        </div>

        {/* mobile trigger */}
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            aria-label={tLanding("nav.menuToggle")}
            onClick={() => setMobileOpen(true)}
          >
            <Menu />
          </Button>
          <SheetContent side="right" className="w-full sm:max-w-xs">
            <SheetHeader>
              <SheetTitle>{t("appName")}</SheetTitle>
            </SheetHeader>
            <div className="flex flex-col gap-1 px-4">
              <p className="px-1 pb-1 text-xs font-medium text-muted-foreground">
                {tLanding("nav.explore")}
              </p>
              {exploreItems.map((item) => (
                <SheetClose
                  key={item.title}
                  nativeButton={false}
                  render={
                    <a
                      href={item.href}
                      className="flex items-center gap-3 rounded-lg p-2 hover:bg-muted"
                    />
                  }
                >
                  <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <item.icon className="size-4" aria-hidden />
                  </span>
                  <span className="text-sm font-medium">{item.title}</span>
                </SheetClose>
              ))}
            </div>
            <div className="mt-auto flex flex-col gap-3 border-t p-4">
              <div className="flex items-center justify-between">
                <LocaleSwitcher />
                <ThemeToggle />
              </div>
              <SheetClose
                nativeButton={false}
                render={<Link href="/onboarding" />}
                className="text-center text-sm font-medium text-foreground"
              >
                {tLanding("nav.tryDemo")}
              </SheetClose>
              <SheetClose
                nativeButton={false}
                render={<Link href="/login" />}
                className={cn(
                  "text-center text-sm font-medium text-foreground",
                )}
              >
                {tAuth("login")}
              </SheetClose>
              <Button
                nativeButton={false}
                render={<Link href="/register" />}
                className="w-full"
              >
                {tLanding("nav.getStarted")}
              </Button>
            </div>
          </SheetContent>
        </Sheet>
      </div>
      <motion.div
        className="h-px w-full origin-left bg-border"
        initial={{ scaleX: 0 }}
        animate={{ scaleX: 1 }}
        transition={{ duration: 0.6, ease: "easeOut" }}
      />
    </header>
  );
}
