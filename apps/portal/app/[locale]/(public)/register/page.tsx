import { notFound } from "next/navigation";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { RegisterForm } from "@/components/auth/register-form";

import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default async function RegisterPage({
  params,
}: PageProps<"/[locale]/register">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  const { t } = await getTranslation(locale, "auth");

  return (
    <main className="mx-auto h-screen w-full flex items-center justify-center">
      <Card className="pb-16 pt-12 px-4 w-full xl:max-w-150">
        <CardHeader className="mb-4">
          <h1 className="text-2xl font-semibold tracking-tight">
            {t("register")}
          </h1>
        </CardHeader>
        <CardContent>
          <RegisterForm />
        </CardContent>
      </Card>
    </main>
  );
}
