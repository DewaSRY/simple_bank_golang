import type { Metadata } from "next";
import { LogoutBanner } from "@/feature/auth/components/logout-banner";

// Transient redirect-through page — never worth indexing.
export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default function LogoutPage() {
  return <LogoutBanner />;
}
