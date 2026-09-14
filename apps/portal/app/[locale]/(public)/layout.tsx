export default async function ProtectedLayout({
  children,
}: LayoutProps<"/[locale]">) {
  return (
    <div className="min-h-screen bg-linear-to-b from-muted/60 to-background dark:from-background dark:to-background">
      {children}
    </div>
  );
}
