import type { Metadata } from "next";
import { AuthSuccessFlow } from "@/feature/auth/components/auth-success-flow";

export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default function AuthSuccessPage() {
  return <AuthSuccessFlow />;
}
