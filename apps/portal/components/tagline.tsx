"use client";

import { Trans } from "react-i18next";

export function Tagline() {
  return (
    <Trans
      i18nKey="tagline"
      ns="common"
      components={{
        brand: <span className="text-zinc-500 dark:text-zinc-400" />,
      }}
    />
  );
}
