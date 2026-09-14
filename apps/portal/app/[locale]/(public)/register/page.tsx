import { notFound } from "next/navigation";
import { Wallet } from "lucide-react";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { RegisterForm } from "@/components/auth/register-form";
import { Link } from "@/i18n/navigation";

import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default async function RegisterPage({
  params,
}: PageProps<"/[locale]/register">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  const { t } = await getTranslation(locale, "auth");
  const { t: tCommon } = await getTranslation(locale, "common");

  return (
    <main className="mx-auto flex min-h-screen w-full items-center justify-center px-4 py-12">
      <div className="flex w-full flex-col items-center gap-6 sm:max-w-100">
        <Link
          href="/"
          className="flex items-center gap-2 text-lg font-semibold tracking-tight"
        >
          <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Wallet className="size-4" aria-hidden />
          </span>
          {tCommon("appName")}
        </Link>

        <Card className="w-full py-10 sm:px-2">
          <CardHeader className="mb-2 text-center">
            <h1 className="text-2xl font-semibold tracking-tight">
              {t("register")}
            </h1>
          </CardHeader>
          <CardContent>
            <RegisterForm />
          </CardContent>
        </Card>
      </div>
    </main>
  );
}
