export default async function ProtectedLayout({
  children,
}: LayoutProps<"/[locale]">) {
  return <div className="bg-zinc-50 dark:bg-gray-900">{children}</div>;
}
