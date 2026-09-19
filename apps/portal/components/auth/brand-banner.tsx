import { Wallet } from "lucide-react";

import { useTranslation } from "react-i18next";
import { GuardedLink } from "../common/navigation-guard/guarded-link";

export function BrandBanner() {
  const { t: tCommon } = useTranslation("common");

  return (
    <GuardedLink
      href="/"
      className="flex items-center gap-2 text-lg font-semibold tracking-tight"
    >
      <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <Wallet className="size-4" aria-hidden />
      </span>
      {tCommon("appName")}
    </GuardedLink>
  );
}
