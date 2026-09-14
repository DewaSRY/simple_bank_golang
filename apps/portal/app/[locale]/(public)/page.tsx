import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { MotionConfig } from "motion/react";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { LandingNav } from "@/components/landing/landing-nav";
import { HeroSection } from "@/components/landing/hero-section";
import { ProblemSection } from "@/components/landing/problem-section";
import { SolutionSection } from "@/components/landing/solution-section";
import { FeaturesSection } from "@/components/landing/features-section";
import { HowItWorksSection } from "@/components/landing/how-it-works-section";
import { FinalCtaSection } from "@/components/landing/final-cta-section";
import { LandingFooter } from "@/components/landing/landing-footer";

export async function generateMetadata({
  params,
}: PageProps<"/[locale]">): Promise<Metadata> {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    return {};
  }

  const { t } = await getTranslation(locale, "common");
  const title = t("landing.metaTitle");
  const description = t("landing.metaDescription");

  return {
    title,
    description,
    openGraph: {
      title,
      description,
      type: "website",
    },
  };
}

export default async function Home({ params }: PageProps<"/[locale]">) {
  const { locale } = await params;

  if (!isAppLocale(locale)) {
    notFound();
  }

  return (
    <MotionConfig reducedMotion="user">
      <main className="flex min-h-screen w-full flex-col">
        <LandingNav />
        <HeroSection />
        <ProblemSection />
        <SolutionSection />
        <FeaturesSection />
        <HowItWorksSection />
        <FinalCtaSection />
        <LandingFooter />
      </main>
    </MotionConfig>
  );
}
