"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDeleteAccount } from "@/feature/account-manage/hooks/query";
import { getApiErrorMessage } from "@/lib/api/error";
import type { AccountWithUserName } from "@/feature/account/type";

interface Props {
  account: AccountWithUserName;
  open?: boolean;
  setOpen?: (open: boolean) => void;
  onDeleted: () => void;
}

export function DeleteAccountDialog({
  account,
  open,
  setOpen,
  onDeleted,
}: Props) {
  const { mutateAsync, isPending } = useDeleteAccount(account.id);
  const [error, setError] = useState<string | null>(null);

  function handleOpenChange(nextOpen: boolean) {
    setOpen?.(nextOpen);
    if (nextOpen) setError(null);
  }

  async function handleConfirm() {
    setError(null);
    try {
      await mutateAsync();
      handleOpenChange(false);
      onDeleted();
    } catch (err) {
      setError(
        getApiErrorMessage(err, "Unable to delete this account. Please try again."),
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-sm px-4">
        <DialogHeader>
          <DialogTitle>Delete account</DialogTitle>
          <DialogDescription>
            This will permanently delete &ldquo;{account.name}&rdquo; and
            sweep its remaining balance to your main account. This action
            cannot be undone.
          </DialogDescription>
        </DialogHeader>

        {error && <p className="text-sm text-destructive">{error}</p>}

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleConfirm}
            disabled={isPending}
          >
            {isPending ? "Deleting..." : "Delete account"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
