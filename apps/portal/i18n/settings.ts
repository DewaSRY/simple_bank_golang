export const locales = ["en", "id"] as const;
export const defaultLocale: AppLocale = "en";

export type AppLocale = (typeof locales)[number];

export function isAppLocale(value: string): value is AppLocale {
  return (locales as readonly string[]).includes(value);
}

export const namespaces = ["common", "auth", "transfer"] as const;
export type AppNamespace = (typeof namespaces)[number];
export const defaultNamespace: AppNamespace = "common";

export function getI18nOptions(
  locale: AppLocale = defaultLocale,
  ns: AppNamespace | readonly AppNamespace[] = defaultNamespace,
) {
  return {
    supportedLngs: locales,
    fallbackLng: defaultLocale,
    lng: locale,
    fallbackNS: defaultNamespace,
    defaultNS: defaultNamespace,
    ns,
    interpolation: {
      escapeValue: false,
    },
  };
}
