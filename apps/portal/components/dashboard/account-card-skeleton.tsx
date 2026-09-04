"use client";

import { Card } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

export function AccountCardSkeleton() {
  return (
    <ul className="flex flex-col gap-3">
      {Array.from({ length: 4 }).map((_, i) => (
        <li key={i}>
          <Card className="flex-row items-center justify-between px-4">
            <div className="flex items-center gap-3">
              <Skeleton className="size-10 shrink-0 rounded-full" />
              <div className="flex flex-col gap-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-20" />
              </div>
            </div>
            <div className="flex flex-col items-end gap-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-10" />
            </div>
          </Card>
        </li>
      ))}
    </ul>
  );
}
