import { getTranslation } from "@/i18n/server";
import type { AppLocale } from "@/i18n/settings";
import { logoutAction } from "../../feature/auth/actions";
import { Button } from "@/components/ui/button";

export async function LogoutButton({ locale }: { locale: AppLocale }) {
  const { t } = await getTranslation(locale, "auth");

  return (
    <form action={logoutAction.bind(null, locale)}>
      <Button type="submit" variant="outline" size="sm">
        {t("logout")}
      </Button>
    </form>
  );
}
