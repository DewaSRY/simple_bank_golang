"use client";

import { useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { pushToast } from "@/lib/toast/store";

interface Props {
  value: string;
  /** Accessible label, e.g. "Copy account number". */
  label: string;
  className?: string;
}

export function CopyButton({ value, label, className }: Props) {
  const { t } = useTranslation("common");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timeout);
  }, [copied]);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      pushToast({ variant: "success", title: { key: "common:copied" } });
    } catch {
      pushToast({ variant: "error", title: { key: "common:copyFailed" } });
    }
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-xs"
      onClick={handleCopy}
      aria-label={copied ? t("copied") : label}
      className={cn("text-current/70 hover:text-current", className)}
    >
      {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
    </Button>
  );
}
