"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * Guards in-app navigation, the browser back/forward buttons, and tab
 * close/refresh while `when` is true, prompting for confirmation before
 * letting any of them through.
 *
 * - Link/router navigation: wrap the intended action in `guardNavigate` (e.g.
 *   inside a `<Link onNavigate>` handler or before a `router.push` call).
 * - Back/forward: a sentinel history entry is pushed so the first back press
 *   re-triggers this guard instead of leaving the page outright.
 * - Tab close/refresh: falls back to the native `beforeunload` prompt, which
 *   browsers show with their own fixed copy (they ignore custom messages).
 */
export function useUnsavedChangesGuard(when: boolean) {
  const [isOpen, setIsOpen] = useState(false);
  const pendingActionRef = useRef<(() => void) | null>(null);
  const allowNextPopRef = useRef(false);
  const restoringSentinelRef = useRef(false);

  useEffect(() => {
    if (!when) return;

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () =>
      window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [when]);

  useEffect(() => {
    if (!when) return;

    // Push a single sentinel entry so the first back press lands here
    // instead of leaving the page. A cancelled attempt is undone with
    // `history.forward()` (back onto this same entry) rather than pushing
    // another sentinel, so repeated cancellations never pile up history
    // entries that a later confirmed "leave" would then have to unwind.
    window.history.pushState(null, "", window.location.href);

    const handlePopState = () => {
      if (allowNextPopRef.current) {
        allowNextPopRef.current = false;
        return;
      }

      if (restoringSentinelRef.current) {
        restoringSentinelRef.current = false;
        return;
      }

      restoringSentinelRef.current = true;
      window.history.forward();

      pendingActionRef.current = () => {
        allowNextPopRef.current = true;
        // Two steps back: past the sentinel entry we pushed, and past the
        // real entry it duplicated, landing on whatever preceded it.
        window.history.go(-2);
      };
      setIsOpen(true);
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [when]);

  const guardNavigate = useCallback(
    (action: () => void) => {
      if (!when) {
        action();
        return true;
      }

      pendingActionRef.current = action;
      setIsOpen(true);
      return false;
    },
    [when],
  );

  const confirmLeave = useCallback(() => {
    setIsOpen(false);
    const action = pendingActionRef.current;
    pendingActionRef.current = null;
    action?.();
  }, []);

  const cancelLeave = useCallback(() => {
    setIsOpen(false);
    pendingActionRef.current = null;
  }, []);

  return { isOpen, guardNavigate, confirmLeave, cancelLeave };
}

interface UnsavedChangesDialogProps {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function UnsavedChangesDialog({
  open,
  onConfirm,
  onCancel,
}: UnsavedChangesDialogProps) {
  const { t } = useTranslation("common");

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onCancel()}>
      <DialogContent className="sm:max-w-sm px-4">
        <DialogHeader>
          <DialogTitle>{t("unsavedChangesTitle")}</DialogTitle>
          <DialogDescription>
            {t("unsavedChangesDescription")}
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel}>
            {t("cancel")}
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            {t("leave")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
