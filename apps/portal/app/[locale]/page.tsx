import { getTranslations, setRequestLocale } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { ThemeToggle } from "@/components/theme-toggle";
import { Link } from "@/i18n/navigation";

export default async function Home({ params }: PageProps<"/[locale]">) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations("Common");
  const tAuth = await getTranslations("Auth");

  const features = [
    {
      title: t("featureAccountsTitle"),
      description: t("featureAccountsDesc"),
    },
    {
      title: t("featureTransfersTitle"),
      description: t("featureTransfersDesc"),
    },
    {
      title: t("featureSecurityTitle"),
      description: t("featureSecurityDesc"),
    },
  ];

  return (
    <div className="flex flex-col flex-1 items-center justify-center bg-zinc-50 font-sans dark:bg-black">
      <main className="flex flex-1 w-full max-w-3xl flex-col items-center justify-between py-32 px-16 bg-white dark:bg-black sm:items-start">
        <div className="flex w-full items-center justify-between">
          <span className="text-lg font-semibold text-black dark:text-zinc-50">
            {t("appName")}
          </span>
          <div className="flex items-center gap-4">
            <Link
              href="/login"
              className="text-sm font-medium text-zinc-950 dark:text-zinc-50"
            >
              {tAuth("login")}
            </Link>
            <LocaleSwitcher />
            <ThemeToggle />
          </div>
        </div>
        <div className="flex flex-col items-center gap-6 text-center sm:items-start sm:text-left">
          <h1 className="max-w-md text-3xl font-semibold leading-10 tracking-tight text-black dark:text-zinc-50">
            {t.rich("tagline", {
              brand: (chunks) => (
                <span className="text-zinc-500 dark:text-zinc-400">
                  {chunks}
                </span>
              ),
            })}
          </h1>
          <p className="max-w-md text-lg leading-8 text-zinc-600 dark:text-zinc-400">
            {t("cta")}
          </p>
        </div>
        <div className="grid w-full grid-cols-1 gap-6 sm:grid-cols-3">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="flex flex-col gap-1.5 rounded-lg border border-black/[.08] p-4 dark:border-white/[.145]"
            >
              <h2 className="text-sm font-semibold text-black dark:text-zinc-50">
                {feature.title}
              </h2>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
        <div className="flex flex-col gap-4 text-base font-medium sm:flex-row">
          <Link
            href="/register"
            className="flex h-12 w-full items-center justify-center gap-2 rounded-full bg-foreground px-5 text-background transition-colors hover:bg-[#383838] dark:hover:bg-[#ccc] md:w-[158px]"
          >
            {t("getStarted")}
          </Link>
          <Link
            href="/login"
            className="flex h-12 w-full items-center justify-center rounded-full border border-solid border-black/[.08] px-5 transition-colors hover:border-transparent hover:bg-black/[.04] dark:border-white/[.145] dark:hover:bg-[#1a1a1a] md:w-[158px]"
          >
            {tAuth("login")}
          </Link>
        </div>
      </main>
    </div>
  );
}
