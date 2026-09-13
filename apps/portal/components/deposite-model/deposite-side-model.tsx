import { useState } from "react";
import { DepositeDialog } from "./deposite-dialog";
import { IconButton } from "@/components/common/icon-button";
import { Plus } from "lucide-react";

export function DepositeSideModel() {
  const [open, setOpen] = useState(false);

  return (
    <div>
      <DepositeDialog open={open} setOpen={setOpen} />
      <IconButton
        text="Deposite"
        icon={<Plus />}
        tooltip="Deposite funds"
        onClick={() => setOpen(true)}
        iconPosition="right"
      />
    </div>
  );
}
