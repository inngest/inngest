import Link from 'next/link';
import { CheckIcon, ChevronRightIcon } from '@heroicons/react/20/solid';
import clsx from 'clsx';

import Container from '../layout/Container';

/**
 * NOTE - When you update hero copy also update index.tsx's getStaticProps title/description for social & SEO
 */
export default function Hero() {
  return (
    <Container className="mt-12">
      <div className="mx-auto flex max-w-7xl flex-col justify-between gap-16 md:flex-row md:gap-24">
        <div className="mb-12 mt-12 max-w-[580px] md:mt-24">
          <h1
            className="bg-gradient-to-br from-white to-slate-300 bg-clip-text pb-8 text-4xl font-semibold tracking-tight text-transparent md:text-5xl"
            style={
              {
                WebkitTextStroke: '0.4px #ffffff80',
                WebkitTextFillColor: 'transparent',
                textShadow: '-1px -1px 0 hsla(0,0%,100%,.2), 1px 1px 0 rgba(0,0,0,.1)',
              } as any
            } // silence the experimental webkit props
          >
            {/* Build reliable products */}
            Effortless serverless queues, background jobs, and workflows
          </h1>
          <div className="flex flex-col gap-6 text-base font-normal md:text-lg">
            <p>
              Easily develop serverless workflows in your current codebase, without any new
              infrastructure.
            </p>
            <ul className="flex flex-col gap-2">
              {[
                'Run on serverless, servers or edge',
                'Zero-infrastructure to manage',
                'Automatic retries for max reliability',
              ].map((r) => (
                <li className="flex items-center gap-2" key={r}>
                  <CheckIcon className="h-5 w-5 shrink-0 text-slate-400/80" /> {r}
                </li>
              ))}
            </ul>
            <p>
              Inngest's{' '}
              <Link
                href="/blog/how-durable-workflow-engines-work?ref=homepage-hero"
                className="text-indigo-200 underline decoration-slate-50/50 decoration-dotted underline-offset-2 transition hover:text-indigo-300"
              >
                durable workflow platform
              </Link>{' '}
              and SDKs enable your entire team to ship reliable products.
            </p>
            <div className="flex flex-wrap gap-4 pt-8 text-base">
              <div>
                <Link
                  href="/docs?ref=homepage-hero"
                  className="group flex flex-row items-center whitespace-nowrap rounded-md bg-indigo-500 px-6 py-2 font-medium text-white transition-all hover:bg-indigo-400"
                >
                  Quick Start Guide{' '}
                  <ChevronRightIcon className="relative top-px h-5 transition-transform duration-150 group-hover:translate-x-1" />
                </Link>
              </div>
              <Link
                href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=homepage-hero`}
                className="whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-6 py-2 font-medium text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
              >
                Start Building For Free
              </Link>
            </div>
          </div>
        </div>
        <div className="flex items-center justify-items-center bg-[url(/assets/homepage/hero-paths-graphic.svg)] bg-contain bg-center	bg-no-repeat tracking-tight">
          <div className="text-md m-auto grid border-collapse overflow-hidden rounded-lg border border-slate-100/10 font-medium backdrop-blur-sm lg:min-w-[460px] lg:grid-cols-2">
            {[
              'Serverless queues',
              'Background jobs',
              'Durable workflows',
              'AI & LLM chaining',
              'Custom workflow engines',
              'Webhook event processing',
            ].map((t, idx, a) => (
              <div
                className={clsx(
                  'min-w-[220px] whitespace-nowrap border border-slate-100/10 px-3 py-3 shadow-lg',
                  idx === 0 && 'rounded-t-md lg:rounded-tr-none',
                  idx === 1 && 'lg:rounded-tr-md',
                  idx === a.length - 2 && 'lg:rounded-bl-md',
                  idx === a.length - 1 && 'rounded-b-md lg:rounded-bl-none'
                )}
              >
                {t}
              </div>
            ))}
          </div>
        </div>
      </div>
    </Container>
  );
}
