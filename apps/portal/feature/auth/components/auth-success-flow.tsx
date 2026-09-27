"use client";

import { useState } from "react";
import { AuthSuccessBanner } from "./auth-success-banner";
import { AuthAccountPreview } from "./auth-account-preview";

export function AuthSuccessFlow() {
  const [showPreview, setShowPreview] = useState(false);

  if (!showPreview) {
    return <AuthSuccessBanner onComplete={() => setShowPreview(true)} />;
  }

  return <AuthAccountPreview />;
}
