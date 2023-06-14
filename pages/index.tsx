import Link from "next/link";
import React from "react";
import Header from "../shared/Header";
import Hero from "../shared/Home/Hero";
import Logos from "src/shared/Home/Logos";
import SDKOverview from "src/shared/Home/SDKOverview";

import LocalDev from "../shared/Home/LocalDev";
import OutTheBox from "../shared/Home/OutTheBox";
import FullyManaged from "../shared/Home/FullyManaged";
import SocialCTA from "../shared/Home/SocialCTA";
import Footer from "../shared/Footer";
import CustomerQuote from "src/shared/Home/CustomerQuote";

import Patterns from "src/shared/Home/Patterns";
import GetThingsShipped from "src/shared/Home/GetThingsShipped";
import RunAnywhere from "src/shared/Home/RunAnywhere";
import PlatformFeatures from "src/shared/Home/PlatformFeatures";
import FeatureCallouts from "src/shared/Home/FeatureCallouts";

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Effortless serverless queues, background jobs, and workflows",
        description:
          "Easily develop serverless workflows in your current codebase, without any new infrastructure. Using Inngest, your entire team can ship reliable products.",
      },
    },
  };
}

export default function Home() {
  return (
    <div className="home font-sans bg-[#050911]">
      <Header />
      <div
        style={{
          backgroundImage: `radial-gradient(63.13% 57.7% at 50% 33.33%, #0F003C 0%, rgba(5, 9, 17, 0) 100%)`,
        }}
      >
        <Hero />
      </div>

      <Logos
        heading="Trusted by teams all over the world"
        logos={[
          { src: "/assets/customers/ocoya.svg", name: "Ocoya" },
          { src: "/assets/customers/snaplet-dark.svg", name: "Snaplet" },
          { src: "/assets/customers/tono-logo.png", name: "Tono Health" },
          // { src: "/assets/customers/semgrep-logo.svg", name: "Semgrep" },
          { src: "/assets/customers/finta-logo.png?v=1", name: "Finta.io" },
          { src: "/assets/customers/yoke-logo.svg", name: "Yoko" },
        ]}
      />

      <div
        style={{
          background: `url(/assets/textures/diagonal-cross.svg) no-repeat 0 160%`,
          backgroundSize: "cover",
        }}
      >
        <SDKOverview />

        <Logos
          heading="Use your existing framework (or no framework!)"
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

      <LocalDev className="-mb-96" />

      <div className="bg-white pt-96 pb-48">
        <div
          style={{
            backgroundImage: "url(/assets/pricing/table-bg.png)",
            backgroundPosition: "center -30px",
            backgroundRepeat: "no-repeat",
            backgroundSize: "1800px 1200px",
          }}
          className="w-full h-100"
        ></div>

        <RunAnywhere />

        <Logos
          heading={
            <>
              Your code runs your existing platform, or{" "}
              <Link
                href="/docs/deploy?ref=homepage-platforms"
                className="text-slate-700 underline hover:text-slate-900"
              >
                anywhere you choose
              </Link>
              :
            </>
          }
          logos={[
            {
              src: "/assets/brand-logos/vercel-dark.svg",
              name: "Vercel",
              href: "/docs/deploy/vercel?ref=homepage-platforms",
            },
            {
              src: "/assets/brand-logos/netlify-dark.svg",
              name: "Netlify",
              href: "/docs/deploy/netlify?ref=homepage-platforms",
            },
            {
              src: "/assets/brand-logos/cloudflare-dark.svg",
              name: "Cloudflare Pages",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-cloudflare",
            },
            {
              src: "/assets/brand-logos/aws-dark.svg",
              name: "AWS Lambda",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-aws-lambda",
            },
            {
              src: "/assets/brand-logos/google-cloud-dark.svg",
              name: "Google Cloud Functions",
              href: "/docs/sdk/serve?ref=homepage-frameworks#framework-google-cloud-functions",
            },
          ]}
          variant="light"
        />

        <CustomerQuote
          quote="We switched from our PostgreSQL backed queue to Inngest in less than a day. Their approach is idiomatic with a great developer experience. Inngest allowed us to stop worrying about scalability and stability."
          name="Peter Pistorius - CEO @ Snaplet"
          avatar="/assets/customers/snaplet-peter-pistorius.png"
          className="mx-auto mb-24 max-w-2xl"
          variant="light"
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

      {/* <Roadmap /> */}

      <SocialCTA />

      <Footer />
    </div>
  );
}
