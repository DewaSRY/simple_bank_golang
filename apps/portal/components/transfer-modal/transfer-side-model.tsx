import { useState } from "react";
import { TransferDialog } from "./transfer-dialog";
import { IconButton } from "@/components/common/icon-button";
import { Plus } from "lucide-react";

export function TransferSideModel() {
  const [open, setOpen] = useState(false);

  return (
    <div>
      <TransferDialog open={open} setOpen={setOpen} />
      <IconButton
        text="Transfer"
        icon={<Plus />}
        tooltip="Transfer funds"
        onClick={() => setOpen(true)}
        iconPosition="right"
      />
    </div>
  );
}
