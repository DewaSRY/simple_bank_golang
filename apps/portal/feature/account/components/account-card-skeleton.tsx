"use client";

import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function AccountCardSkeleton() {
  return (
    <div className="space-y-8" aria-hidden>
      <Skeleton className="h-36 w-full rounded-xl" />
      <div className="space-y-4">
        <Skeleton className="h-5 w-32" />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i} className="gap-4 px-5 py-5!">
              <div className="flex items-center gap-3">
                <Skeleton className="size-10 shrink-0 rounded-lg" />
                <div className="flex flex-1 flex-col gap-2">
                  <Skeleton className="h-4 w-28" />
                  <Skeleton className="h-3 w-20" />
                </div>
              </div>
              <div className="space-y-2 border-t pt-4">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-6 w-36" />
              </div>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}
