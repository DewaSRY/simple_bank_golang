import { notFound } from "next/navigation";
import { isAppLocale } from "@/i18n/settings";
import { LoginFormScreen } from "@/components/auth/login-form-screen";

export default async function LoginPage({
  params,
}: PageProps<"/[locale]/login">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  return <LoginFormScreen />;
}
