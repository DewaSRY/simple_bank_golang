import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { MotionConfig } from "motion/react";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { canonicalFor, buildLanguageAlternates } from "@/lib/seo/metadata";
import { LandingNav } from "@/components/landing/landing-nav";
import { HeroSection } from "@/components/landing/hero-section";
import { ProblemSection } from "@/components/landing/problem-section";
import { SolutionSection } from "@/components/landing/solution-section";
import { FeaturesSection } from "@/components/landing/features-section";
import { ArchitectureSection } from "@/components/landing/architecture-section";
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
    title: { absolute: title },
    description,
    alternates: {
      canonical: canonicalFor(locale, ""),
      languages: buildLanguageAlternates(""),
    },
    openGraph: {
      title,
      description,
      type: "website",
      url: canonicalFor(locale, ""),
      locale,
      images: [
        {
          url: "/icons/android-chrome-512x512.png",
          width: 512,
          height: 512,
          alt: title,
        },
      ],
    },
    twitter: {
      card: "summary",
      title,
      description,
      images: ["/icons/android-chrome-512x512.png"],
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: "Simple Bank",
            url: canonicalFor(locale, ""),
          }),
        }}
      />
      <main className="flex min-h-screen w-full flex-col">
        <LandingNav />
        <HeroSection />
        <ProblemSection />
        <SolutionSection />
        <FeaturesSection />
        <ArchitectureSection />
        <HowItWorksSection />
        <FinalCtaSection />
        <LandingFooter />
      </main>
    </MotionConfig>
  );
}
