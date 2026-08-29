import { hasLocale } from "next-intl";
import { getRequestConfig } from "next-intl/server";
import { routing } from "./routing";

export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;

  const [common, auth, transfer] = await Promise.all([
    import(`../messages/${locale}/common.json`),
    import(`../messages/${locale}/auth.json`),
    import(`../messages/${locale}/transfer.json`),
  ]);

  return {
    locale,
    messages: {
      Common: common.default,
      Auth: auth.default,
      Transfer: transfer.default,
    },
  };
});
