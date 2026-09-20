import type { Metadata } from "next";
import { AuthSuccessBanner } from "@/components/auth/auth-success-banner";

export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default function AuthSuccessPage() {
  return <AuthSuccessBanner />;
}
