"use client";
import { useState } from "react";
import { SidebarGroup, SidebarGroupContent } from "@/components/ui/sidebar";

import { formatBalance } from "@/lib/number";
import { useAccounts } from "@/feature/account/hooks/query";
import { Button } from "@/components/ui/button";

import { CreateAccountDialog } from "@/components/navigation/create-account-dialog";
import { Plus } from "lucide-react";
import Link from "next/link";

export function NavAccountList() {
  const { data: accounts, isLoading } = useAccounts();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  return (
    <SidebarGroup>
      <CreateAccountDialog open={isDialogOpen} setOpen={setIsDialogOpen} />

      <SidebarGroupContent className="flex flex-col gap-2">
        <div className="mb-2 flex justify-between">
          <h2 className="text-lg font-bold">Accounts</h2>
          <div>
            <Button variant="outline" onClick={() => setIsDialogOpen(true)}>
              <Plus />
            </Button>
          </div>
        </div>

        <div>
          {accounts?.map((account) => (
            <Link
              key={account.id}
              className="min-h-min! py-2 "
              role="button"
              href={`/account/${account.id}`}
            >
              <div className="flex flex-col justify-between ">
                <h2 className="text-sm ">{account.name || account.username}</h2>
                <div>
                  <span className="text-right text-xs text-gray-500">
                    {account.number}
                  </span>
                </div>
              </div>

              <div className="flex justify-end text-xs  ">
                <span>{formatBalance(account.balance)}</span>
                <span className="ml-0.5">{account.currency}</span>
              </div>
            </Link>
          ))}
        </div>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
