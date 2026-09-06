"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AccountWithUserName } from "@/feature/account/client";

export function AccountDetailsCard({
  account,
}: {
  account: AccountWithUserName;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Account details</CardTitle>
        <CardDescription>Information for this account</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="text-xs text-muted-foreground">Currency</p>
          <p className="mt-1 font-medium">{account.currency}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Description</p>
          <p className="mt-1 font-medium">
            {account.description || "No description"}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
