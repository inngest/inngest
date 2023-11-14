import Link from "next/link";
import Image from "next/image";
import Container from "../layout/Container";
import Logos from "src/shared/Home/Logos";

import Heading from "./Heading";

export default function Flexibility() {
  return (
    <Container className="my-44 tracking-tight">
      <Heading
        title="Flexibility for your team"
        lede={
          <>
            Use Inngest where, how and with whatever you want. Flexible and
            extensible for all teams.
          </>
        }
        className="mx-auto max-w-3xl text-center"
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 p-px gap-px mt-16 mx-auto max-w-6xl rounded-md bg-gradient-to-tl from-green-800/60 via-orange-300/60 to-rose-900/60 shadow-[0_10px_100px_0_rgba(52,211,153,0.15)]">
        <div className="lg:col-span-2 p-8 md:rounded-t-md bg-slate-1000">
          <SectionHeading className="text-xl md:text-4xl mt-6 lg:mt-12">
            Works in any cloud
          </SectionHeading>
          <p className="max-w-xl mx-auto my-6 text-center text-lg mb-7 font-medium text-slate-300">
            Run your Inngest functions, securely, on your own cloud, wherever
            that may be. Inngest calls you, so all you need as a URL and we take
            care of the rest.
          </p>
          <Logos
            className="px-0 md:px-0 lg:px-0 my-12 lg:my-12 xl:mb-12"
            logos={[
              {
                src: "/assets/brand-logos/vercel-white.svg",
                name: "Vercel",
                href: "/docs/deploy/vercel?ref=homepage-platforms",
              },
              {
                src: "/assets/brand-logos/netlify-logo.svg",
                name: "Netlify",
                href: "/docs/deploy/netlify?ref=homepage-platforms",
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
              {
                src: "/assets/brand-logos/cloudflare-white.svg",
                name: "Cloudflare Pages",
                href: "/docs/sdk/serve?ref=homepage-frameworks#framework-cloudflare",
              },
            ]}
          />
        </div>
        <div className="md:rounded-bl-md p-8 bg-slate-1000">
          <SectionHeading className="text-xl md:text-2xl mt-6">
            Drop into your codebase
          </SectionHeading>
          <p className="max-w-xl mx-auto px-6 my-6 text-center text-lg mb-7 font-medium text-slate-300">
            Our framework adapters make it easy to get to production quickly.
          </p>
          <div className="my-12 grid grid-cols-3 gap-6 gap-y-8 items-center justify-items-center">
            {[
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
                src: "/assets/brand-logos/remix-white.svg",
                name: "Remix",
                href: "/docs/sdk/serve?ref=homepage-frameworks#framework-remix",
              },
              {
                src: "/assets/brand-logos/cloudflare-white.svg",
                name: "Cloudflare Pages",
                href: "/docs/sdk/serve?ref=homepage-frameworks#framework-cloudflare",
              },
              {
                src: "/assets/brand-logos/redwoodjs-white.svg",
                name: "RedwoodJS",
                href: "/docs/sdk/serve?ref=homepage-frameworks#framework-redwood",
              },
              {
                src: "/assets/brand-logos/fastify-white.svg",
                name: "Fastify",
                href: "/docs/sdk/serve?ref=homepage-frameworks#framework-fastify",
              },
              // FUTURE - Add other language frameworks like Flask
            ].map(({ src, name, href }, idx) => (
              <Link href={href} className="group">
                <Image
                  key={idx}
                  src={src}
                  alt={name}
                  width={120}
                  height={30}
                  className="text-white max-h-[40px] pointer-events-none opacity-80 group:hover:opacity-100 transition-all group-hover:opacity-100 grayscale group-hover:grayscale-0"
                />
              </Link>
            ))}
          </div>

          <div className="flex items-center">
            <Link
              href="/docs/sdk/serve?ref=homepage-frameworks"
              className="mx-auto rounded-md font-medium px-6 py-2 bg-transparent transition-all text-white border border-slate-800 hover:border-slate-600 hover:bg-slate-500/10 whitespace-nowrap"
            >
              Find your framework →
            </Link>
          </div>
        </div>
        <div className="md:rounded-br-md p-8 bg-slate-1000">
          <SectionHeading className="text-xl md:text-2xl mt-6">
            Language agnostic
          </SectionHeading>
          <p className="max-w-xl mx-auto px-6 my-6 text-center text-lg mb-7 font-medium text-slate-300">
            From TypeScript and beyond. Inngest is designed to work with any
            backend.
          </p>
          <div className="my-12 grid grid-cols-2 gap-6 gap-y-12 items-center justify-items-center">
            {[
              {
                src: "/assets/brand-logos/typescript.svg",
                name: "TypeScript",
                size: { h: 60, w: 60 },
              },
              {
                src: "/assets/brand-logos/go-logo-blue.svg",
                name: "Go",
                size: { h: 60, w: 120 },
                release: `Q1 2024`,
              },
              {
                src: "/assets/brand-logos/python-logo-only.svg",
                name: "Python",
                size: { h: 60, w: (60 / 101) * 84 },
                release: `Q1 2024`,
              },
              {
                src: "/assets/brand-logos/rust-logo.png",
                name: "Rust",
                size: { h: 60, w: 60 },
                release: `Q2 2024`,
              },
              // FUTURE - Add other language frameworks like Flask
            ].map(({ src, name, size: { h, w }, release }, idx) => (
              <div className="group relative h-full w-full sm:w-auto flex items-center justify-center">
                <Image
                  key={idx}
                  src={src}
                  alt={name}
                  width={w}
                  height={h}
                  className="max-h-[60px] pointer-events-none transition-all"
                />
                {!!release && (
                  <span className="absolute -bottom-2 sm:-bottom-1 -right-6 md:-right-12 px-3 py-0.5 bg-slate-700 text-slate-50 text-xs font-semibold drop-shadow rounded-full whitespace-nowrap">
                    {release}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </Container>
  );
}

function SectionHeading({ children, className = "" }) {
  return (
    <h3
      className={`text-center leading-snug font-semibold tracking-tight bg-gradient-to-br from-white to-slate-300 bg-clip-text text-transparent ${className}`}
    >
      {children}
    </h3>
  );
}
