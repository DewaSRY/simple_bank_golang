import { useState } from "react";

import { IconButton } from "@/components/common/icon-button";
import { CreateAccountDialog } from "@/components/navigation/create-account-dialog";
import { Plus } from "lucide-react";

export function CreateNewAccount() {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  return (
    <div>
      <CreateAccountDialog open={isDialogOpen} setOpen={setIsDialogOpen} />
      <IconButton
        text="Create new account"
        icon={<Plus />}
        tooltip="Create new account"
        onClick={() => setIsDialogOpen(true)}
        iconPosition="right"
      />
    </div>
  );
}
