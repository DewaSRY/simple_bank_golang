import { notFound } from "next/navigation";
import { Wallet, ArrowLeftRight, ShieldCheck } from "lucide-react";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { ThemeToggle } from "@/components/theme-toggle";
import { Tagline } from "@/components/tagline";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

export default async function Home({ params }: PageProps<"/[locale]">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  const { t } = await getTranslation(locale, "common");
  const { t: tAuth } = await getTranslation(locale, "auth");

  const features = [
    {
      icon: Wallet,
      title: t("featureAccountsTitle"),
      description: t("featureAccountsDesc"),
    },
    {
      icon: ArrowLeftRight,
      title: t("featureTransfersTitle"),
      description: t("featureTransfersDesc"),
    },
    {
      icon: ShieldCheck,
      title: t("featureSecurityTitle"),
      description: t("featureSecurityDesc"),
    },
  ];

  return (
    <main className="mx-auto flex min-h-screen w-full items-center justify-center px-4 py-12 sm:px-6">
      <div className="flex w-full max-w-3xl flex-col items-center gap-y-10 rounded-2xl bg-card px-6 py-12 text-card-foreground ring-1 ring-foreground/10 sm:items-start sm:px-16">
        {/* header  */}
        <div className="flex w-full items-center justify-between">
          <span className="flex items-center gap-2 text-lg font-semibold tracking-tight">
            <span className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Wallet className="size-4" aria-hidden />
            </span>
            {t("appName")}
          </span>
          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="text-sm font-medium text-foreground hover:text-foreground/80"
            >
              {tAuth("login")}
            </Link>
            <LocaleSwitcher />
            <ThemeToggle />
          </div>
        </div>

        <div className="flex flex-col items-center gap-3 text-center sm:items-start sm:text-left">
          <h1 className="max-w-md text-4xl font-semibold leading-tight tracking-tight text-balance">
            <Tagline />
          </h1>
          <p className="max-w-md text-lg leading-8 text-muted-foreground text-balance">
            {t("cta")}
          </p>
        </div>

        {/* features list  */}
        <div className="grid w-full grid-cols-1 gap-4 sm:grid-cols-3">
          {features.map((feature) => (
            <Card
              key={feature.title}
              className="gap-2 bg-muted/40 p-4 ring-1 ring-foreground/5"
            >
              <feature.icon className="size-5 text-primary" aria-hidden />
              <h2 className="text-sm font-semibold">{feature.title}</h2>
              <p className="text-sm text-muted-foreground">
                {feature.description}
              </p>
            </Card>
          ))}
        </div>

        <div className="flex w-full flex-col gap-3 sm:flex-row">
          <Button
            size="lg"
            className="h-12 flex-1 text-base"
            nativeButton={false}
            render={<Link href="/register" />}
          >
            {t("getStarted")}
          </Button>
          <Button
            size="lg"
            variant="outline"
            className="h-12 flex-1 text-base"
            nativeButton={false}
            render={<Link href="/login" />}
          >
            {tAuth("login")}
          </Button>
        </div>
      </div>
    </main>
  );
}
