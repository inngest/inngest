import Image from 'next/image';
import Link from 'next/link';
import Logos from 'src/shared/Home/Logos';

import Container from '../layout/Container';
import Heading from './Heading';

export default function Flexibility() {
  return (
    <Container className="my-44 tracking-tight">
      <Heading
        title="Flexibility for your team"
        lede={
          <>
            Use Inngest where, how and with whatever you want. Flexible and extensible for all
            teams.
          </>
        }
        className="mx-auto max-w-3xl text-center"
      />

      <div className="mx-auto mt-16 grid max-w-6xl grid-cols-1 gap-px rounded-md bg-gradient-to-tl from-green-800/60 via-orange-300/60 to-rose-900/60 p-px shadow-[0_10px_100px_0_rgba(52,211,153,0.15)] lg:grid-cols-2">
        <div className="bg-slate-1000 p-8 md:rounded-t-md lg:col-span-2">
          <SectionHeading className="mt-6 text-xl md:text-4xl lg:mt-12">
            Works in any cloud
          </SectionHeading>
          <p className="mx-auto my-6 mb-7 max-w-xl text-center text-lg font-medium text-slate-300">
            Run your Inngest functions, securely, on your own cloud, wherever that may be. Inngest
            calls you, so all you need as a URL and we take care of the rest.
          </p>
          <Logos
            className="my-12 px-0 md:px-0 lg:my-12 lg:px-0 xl:mb-12"
            logos={[
              {
                src: '/assets/brand-logos/vercel-white.svg',
                name: 'Vercel',
                href: '/docs/deploy/vercel?ref=homepage-platforms',
              },
              {
                src: '/assets/brand-logos/netlify-logo.svg',
                name: 'Netlify',
                href: '/docs/deploy/netlify?ref=homepage-platforms',
              },
              {
                src: '/assets/brand-logos/aws-white.svg',
                name: 'AWS Lambda',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-aws-lambda',
              },
              {
                src: '/assets/brand-logos/google-cloud-white.svg',
                name: 'Google Cloud Functions',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-google-cloud-functions',
              },
              {
                src: '/assets/brand-logos/cloudflare-white.svg',
                name: 'Cloudflare Pages',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-cloudflare',
              },
            ]}
          />
        </div>
        <div className="bg-slate-1000 p-8 md:rounded-bl-md">
          <SectionHeading className="mt-6 text-xl md:text-2xl">
            Drop into your codebase
          </SectionHeading>
          <p className="mx-auto my-6 mb-7 max-w-xl px-6 text-center text-lg font-medium text-slate-300">
            Our framework adapters make it easy to get to production quickly.
          </p>
          <div className="my-12 grid grid-cols-3 items-center justify-items-center gap-6 gap-y-8">
            {[
              {
                src: '/assets/brand-logos/next-js-white.svg',
                name: 'Next.js',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-next-js',
              },
              {
                src: '/assets/brand-logos/express-js-white.svg',
                name: 'Express.js',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-express',
              },
              {
                src: '/assets/brand-logos/remix-white.svg',
                name: 'Remix',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-remix',
              },
              {
                src: '/assets/brand-logos/cloudflare-white.svg',
                name: 'Cloudflare Pages',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-cloudflare',
              },
              {
                src: '/assets/brand-logos/redwoodjs-white.svg',
                name: 'RedwoodJS',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-redwood',
              },
              {
                src: '/assets/brand-logos/fastify-white.svg',
                name: 'Fastify',
                href: '/docs/sdk/serve?ref=homepage-frameworks#framework-fastify',
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
                  className="group:hover:opacity-100 pointer-events-none max-h-[40px] text-white opacity-80 grayscale transition-all group-hover:opacity-100 group-hover:grayscale-0"
                />
              </Link>
            ))}
          </div>

          <div className="flex items-center">
            <Link
              href="/docs/sdk/serve?ref=homepage-frameworks"
              className="mx-auto whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-6 py-2 font-medium text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
            >
              Find your framework →
            </Link>
          </div>
        </div>
        <div className="bg-slate-1000 p-8 md:rounded-br-md">
          <SectionHeading className="mt-6 text-xl md:text-2xl">Language agnostic</SectionHeading>
          <p className="mx-auto my-6 mb-7 max-w-xl px-6 text-center text-lg font-medium text-slate-300">
            From TypeScript and beyond. Inngest is designed to work with any backend.
          </p>
          <div className="my-12 grid grid-cols-2 items-center justify-items-center gap-6 gap-y-12">
            {[
              {
                src: '/assets/brand-logos/typescript.svg',
                name: 'TypeScript',
                size: { h: 60, w: 60 },
              },
              {
                src: '/assets/brand-logos/go-logo-blue.svg',
                name: 'Go',
                size: { h: 60, w: 120 },
                release: `Q1 2024`,
              },
              {
                src: '/assets/brand-logos/python-logo-only.svg',
                name: 'Python',
                size: { h: 60, w: (60 / 101) * 84 },
                release: `Q1 2024`,
              },
              {
                src: '/assets/brand-logos/rust-logo.png',
                name: 'Rust',
                size: { h: 60, w: 60 },
                release: `Q2 2024`,
              },
              // FUTURE - Add other language frameworks like Flask
            ].map(({ src, name, size: { h, w }, release }, idx) => (
              <div className="group relative flex h-full w-full items-center justify-center sm:w-auto">
                <Image
                  key={idx}
                  src={src}
                  alt={name}
                  width={w}
                  height={h}
                  className="pointer-events-none max-h-[60px] transition-all"
                />
                {!!release && (
                  <span className="absolute -bottom-2 -right-6 whitespace-nowrap rounded-full bg-slate-700 px-3 py-0.5 text-xs font-semibold text-slate-50 drop-shadow sm:-bottom-1 md:-right-12">
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

function SectionHeading({ children, className = '' }) {
  return (
    <h3
      className={`bg-gradient-to-br from-white to-slate-300 bg-clip-text text-center font-semibold leading-snug tracking-tight text-transparent ${className}`}
    >
      {children}
    </h3>
  );
}
