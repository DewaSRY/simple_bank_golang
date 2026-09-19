import { notFound } from "next/navigation";
import { isAppLocale } from "@/i18n/settings";
import { RegisterFormScreen } from "@/components/auth/register-form-screen";

export default async function RegisterPage({
  params,
}: PageProps<"/[locale]/register">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  return <RegisterFormScreen />;
}
