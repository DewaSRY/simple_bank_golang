import { getTranslation } from "@/i18n/server";
import type { AppLocale } from "@/i18n/settings";
import { logoutAction } from "../../feature/auth/actions";

export async function LogoutButton({ locale }: { locale: AppLocale }) {
  const { t } = await getTranslation(locale, "auth");

  return (
    <form action={logoutAction.bind(null, locale)}>
      <button
        type="submit"
        className="rounded-full border border-solid border-black/80 px-4 py-2 text-sm font-medium transition-colors hover:border-transparent hover:bg-black/[.04] dark:border-white/[.145] dark:hover:bg-[#1a1a1a]"
      >
        {t("logout")}
      </button>
    </form>
  );
}
