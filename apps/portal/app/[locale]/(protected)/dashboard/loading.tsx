import { Skeleton } from "@/components/ui/skeleton";
import { AccountCardSkeleton } from "@/components/dashboard/account-card-skeleton";

export default function DashboardLoading() {
  return (
    <div className="flex my-2 flex-1 flex-col bg-zinc-50 px-6 font-sans dark:bg-black">
      <div className="mx-auto flex w-full items-center justify-between">
        <Skeleton className="h-8 w-40" />
      </div>

      <div className="mt-4">
        <AccountCardSkeleton />
      </div>
    </div>
  );
}
