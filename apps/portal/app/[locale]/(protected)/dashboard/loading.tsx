import { Skeleton } from "@/components/ui/skeleton";
import { PageContainer } from "@/components/common/page-header";
import { AccountCardSkeleton } from "@/feature/account/components/account-card-skeleton";

export default function DashboardLoading() {
  return (
    <PageContainer>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          <Skeleton className="h-8 w-40" />
          <Skeleton className="h-4 w-64" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-9 w-28" />
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-9 w-24" />
        </div>
      </div>
      <AccountCardSkeleton />
    </PageContainer>
  );
}
