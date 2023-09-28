import type { GetStaticPropsResult } from "next";
import Link from "next/link";
import React from "react";
import Header from "../shared/Header";
import Hero from "../shared/Home/Hero";
import Logos from "src/shared/Home/Logos";
import SDKOverview from "src/shared/Home/SDKOverview";

import LocalDev from "../shared/Home/LocalDev";
import SocialCTA from "../shared/Home/SocialCTA";
import Footer from "../shared/Footer";
import CustomerQuote from "src/shared/Home/CustomerQuote";
import SocialProof from "src/shared/Home/SocialProof";

import GetThingsShipped from "src/shared/Home/GetThingsShipped";
import RunAnywhere from "src/shared/Home/RunAnywhere";
import PlatformFeatures from "src/shared/Home/PlatformFeatures";
import FeatureCallouts from "src/shared/Home/FeatureCallouts";
import type { PageProps } from "src/shared/types";

export async function getStaticProps(): Promise<
  GetStaticPropsResult<PageProps>
> {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Effortless serverless queues, background jobs, and workflows",
        description:
          "Easily develop serverless workflows in your current codebase, without any new infrastructure. Using Inngest, your entire team can ship reliable products.",
        image: "/assets/homepage/open-graph.png",
      },
    },
  };
}

export default function Home() {
  return (
    <div
      className="home font-sans bg-slate-1000"
      style={{
        backgroundImage: `radial-gradient(circle at center -20%, #231649 0%, rgba(5, 9, 17, 0) 1500px)`,
        backgroundSize: "100% 1500px",
        backgroundRepeat: "no-repeat",
      }}
    >
      <Header />

      <Hero />

      <Logos
        heading="Trusted by teams all over the world"
        logos={[
          {
            src: "/assets/customers/tripadvisor.svg",
            name: "TripAdvisor",
            featured: true,
          },
          {
            src: "/assets/customers/resend.svg",
            name: "Resend",
            featured: true,
            scale: 0.8,
          },

          {
            src: "/assets/customers/snaplet-dark.svg",
            name: "Snaplet",
          },
          {
            src: "/assets/customers/productlane.svg",
            name: "Productlane",
            scale: 1.3,
          },
          { src: "/assets/customers/ocoya.svg", name: "Ocoya" },
          { src: "/assets/customers/finta-logo.png?v=1", name: "Finta.io" },
        ]}
      />

      <div className="relative">
        <div className="absolute top-80 left-0 right-0 -skew-y-6 bottom-0 bg-gradient-to-b from-slate-900/80 border-t border-slate-800/70 to-slate-1000/0"></div>
        <SDKOverview />

        <Logos
          heading={
            <>
              Use your existing framework (<em>or no framework!</em>)
            </>
          }
          logos={[
            {
              src: "/assets/brand-logos/next-js-white.svg",
              name: "Next.js",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-next-js",
            },
            {
              src: "/assets/brand-logos/express-js-white.svg",
              name: "Express.js",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-express",
            },
            {
              src: "/assets/brand-logos/redwoodjs-white.svg",
              name: "RedwoodJS",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-redwood",
            },
            {
              src: "/assets/brand-logos/remix-white.svg",
              name: "Remix",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-remix",
            },
            {
              src: "/assets/brand-logos/deno-white.svg",
              name: "Deno",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-fresh-deno",
            },
          ]}
        />
      </div>

      <LocalDev className="-mb-80 md:-mb-60" />

      <div className="pt-60 pb-20 md:pb-40">
        <RunAnywhere />

        <Logos
          heading={
            <>
              Your code runs on your existing platform, or{" "}
              <Link
                href="/docs/deploy?ref=homepage-platforms"
                className="underline hover:text-indigo-400"
              >
                anywhere you choose
              </Link>
              :
            </>
          }
          logos={[
            {
              src: "/assets/brand-logos/vercel-white.svg",
              name: "Vercel",
              href: "/docs/deploy/vercel?ref=homepage-platforms",
            },
            {
              src: "/assets/brand-logos/netlify-white.svg",
              name: "Netlify",
              href: "/docs/deploy/netlify?ref=homepage-platforms",
            },
            {
              src: "/assets/brand-logos/cloudflare-white.svg",
              name: "Cloudflare Pages",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-cloudflare",
            },
            {
              src: "/assets/brand-logos/aws-white.svg",
              name: "AWS Lambda",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-aws-lambda",
            },
            {
              src: "/assets/brand-logos/google-cloud-white.svg",
              name: "Google Cloud Functions",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-google-cloud-functions",
            },
          ]}
        />

        <CustomerQuote
          quote="We switched from our PostgreSQL backed queue to Inngest in less than a day. Their approach is idiomatic with a great developer experience. Inngest allowed us to stop worrying about scalability and stability."
          name="Peter Pistorius - CEO @ Snaplet"
          avatar="/assets/customers/snaplet-peter-pistorius.png"
          className="mx-auto mb-28 lg:mb-20 max-w-2xl"
        />
      </div>

      <FeatureCallouts />

      <PlatformFeatures />

      {/* TODO - Add button to link to case study - */}
      <CustomerQuote
        quote="We were struggling with the complexities of managing our social media and e-commerce workflows. Thanks to Inngest, we were able to simplify our development process, speed up our time to market, and deliver a better customer experience. Inngest has become an essential tool in our tech stack."
        name="Aivaras Tumas  - CEO @ Ocoya"
        avatar="/assets/customers/ocoya-aivaras-tumas.png"
        className="mx-auto max-w-2xl"
        cta={{
          href: "/customers/ocoya?ref=homepage",
          text: "Read the Case Study",
        }}
      />

      <GetThingsShipped />

      <SocialProof />

      {/* <Roadmap /> */}

      <SocialCTA />

      <Footer />
    </div>
  );
}
