import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { AccountList } from "./account-list";
interface Props {
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

export function DepositeDialog({ open, setOpen }: Props) {
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="p-4 lg:min-w-4xl">
        <DialogHeader>
          <DialogTitle>Deposite Funds</DialogTitle>
          <DialogDescription>
            Enter the amount you want to deposite. Click save when you&apos;re
            done.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-80 overflow-y-auto">
          <AccountList />
        </div>

        <DialogFooter>
          <DialogClose render={<Button variant="outline">Cancel</Button>} />
          <Button type="submit">Save changes</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
