"use client";

import { Link } from "@/i18n/navigation";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function AccountStateMessage({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <main className="flex flex-1 items-center justify-center bg-zinc-50 px-6 py-12 dark:bg-black">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <Link
            href="/dashboard"
            className="text-sm font-medium underline underline-offset-4"
          >
            Return to dashboard
          </Link>
        </CardContent>
      </Card>
    </main>
  );
}
