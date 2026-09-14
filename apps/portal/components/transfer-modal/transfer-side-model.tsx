import { useState } from "react";
import { useTranslation } from "react-i18next";
import { TransferDialog } from "./transfer-dialog";
import { IconButton } from "@/components/common/icon-button";
import { Plus } from "lucide-react";

export function TransferSideModel() {
  const { t } = useTranslation("transfer");
  const [open, setOpen] = useState(false);

  return (
    <div>
      <TransferDialog open={open} setOpen={setOpen} />
      <IconButton
        text={t("transferAction")}
        icon={<Plus />}
        tooltip={t("transferTooltip")}
        onClick={() => setOpen(true)}
        iconPosition="right"
      />
    </div>
  );
}
