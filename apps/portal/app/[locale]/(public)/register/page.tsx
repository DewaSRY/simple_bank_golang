import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { routing } from "@/i18n/routing";
import { RegisterForm } from "@/components/auth/register-form";

import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default async function RegisterPage({
  params,
}: PageProps<"/[locale]/register">) {
  const { locale } = await params;

  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }

  setRequestLocale(locale);

  const t = await getTranslations("Auth");

  return (
    <main className="mx-auto h-screen w-full flex items-center justify-center">
      <Card className=" py-16 px-4 w-full lg:min-w-150 xl:min-h-200  xl:w-125">
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
