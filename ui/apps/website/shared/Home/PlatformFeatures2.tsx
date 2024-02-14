import Link from 'next/link';
import clsx from 'clsx';

import Replay from '../Icons/Replay';
import Container from '../layout/Container';
import CopyBtn from './CopyBtn';
import Heading from './Heading';

export default function PlatformFeatures() {
  const handleCopyClick = (copy) => {
    navigator.clipboard?.writeText(copy);
  };
  return (
    <Container className="my-44 tracking-tight">
      <Heading
        title="Re-imagined Developer Experience"
        lede={
          <>
            Building and operating code that runs in the background is a pain.
            <br className="hidden md:block" /> Get more done, faster with everything built into our
            platform.
          </>
        }
        className="mx-auto max-w-3xl text-center"
      />

      <div className="mx-auto my-24 grid max-w-6xl grid-cols-1 gap-8 md:grid-cols-2">
        <div className="grid grid-cols-1 items-center gap-8 overflow-hidden rounded-2xl bg-gradient-to-br from-emerald-400/10 to-cyan-400/10 px-8 pt-12 sm:grid-cols-1 md:col-span-2 md:grid-cols-2 md:gap-8 lg:grid-cols-[40%_1fr] lg:gap-24 lg:px-16 lg:pt-0 xl:min-h-[420px]">
          <div className="md:pb-8 lg:py-12 lg:pb-12">
            <h3 className="mb-5 text-2xl font-semibold">Branch environments</h3>
            <p className="mb-7 text-lg font-medium text-slate-300">
              Test your entire application end-to-end with an Inngest environment for every
              development branch that you deploy, without any extra work.
            </p>
            <a
              href="/docs/platform/environments"
              className="mt-4 font-medium text-slate-200 underline decoration-slate-50/30 decoration-dotted underline-offset-4 hover:text-white hover:decoration-white/50"
            >
              Learn more →
            </a>
          </div>

          <div className="flex h-full flex-col justify-end md:pt-8">
            <img
              src="/assets/homepage/branch-envs-screenshot.png"
              alt="Branch environments in the Inngest dashboard"
              className={`
              pointer-events-none
                m-auto -mb-1 w-full max-w-6xl
                max-w-full origin-center rounded-md drop-shadow-sm
                md:min-w-[420px]
                xl:min-w-[520px]
              `}
            />
          </div>
        </div>

        <div className="grid grid-rows-[auto_1fr] gap-8 overflow-hidden rounded-lg bg-gradient-to-b from-amber-400/10	to-rose-400/15 md:gap-10">
          <div className="px-8 pt-11 md:px-10">
            <h3 className="mb-5 text-2xl font-semibold">Real-time observability metrics</h3>
            <p className="mb-7 text-lg font-medium text-slate-300">
              Quickly diagnose system wide issues with built in metrics. View backlogs and spikes in
              failures for every single function. There is no need for instrumenting your code for
              metrics or battling some Cloudwatch dashboard.
            </p>
            <a
              href="/blog/2023-10-27-fn-metrics-release"
              className="mt-4 font-medium text-slate-200 underline decoration-slate-50/30 decoration-dotted underline-offset-4 hover:text-white hover:decoration-white/50"
            >
              Learn more →
            </a>
          </div>
          <div className="flex h-full flex-col justify-end pl-10">
            <img
              src="/assets/homepage/observability-metrics.png"
              alt="Real-time observability metrics"
              className="rounded-tl"
            />
          </div>
        </div>

        <div className="grid grid-rows-[auto_1fr] gap-8 overflow-hidden rounded-lg bg-gradient-to-br from-blue-400/20 to-orange-200/20 md:gap-10">
          <div className="px-8 pt-11 md:px-10">
            <h3 className="mb-5 text-2xl font-semibold">Full logs & history</h3>
            <p className="mb-7 text-lg font-medium text-slate-300">
              Inngest keeps a full history of every event and function run allowing you to easily
              debug any production issues. No more parsing logs or trying to connect the dots over
              workflows that could span days or weeks.
            </p>
            {/* <a
              href="/blog/2023-10-27-fn-metrics-release"
              className="mt-4 font-medium text-slate-200 hover:text-white hover:underline decoration-dotted underline-offset-4 decoration-slate-50/30"
            >
              Learn more →
            </a> */}
          </div>

          <div className="flex h-full flex-col justify-end pl-10">
            <img
              src="/assets/homepage/screenshot-logs-timeline.png"
              alt="Full logs and history"
              className="rounded-tl"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 items-center gap-8 overflow-hidden rounded-2xl bg-gradient-to-bl from-indigo-500/40 to-cyan-200/40 px-8 pt-12 sm:grid-cols-1 md:col-span-2 md:grid-cols-2 md:gap-8 lg:grid-cols-[40%_1fr] lg:gap-24 lg:px-16 lg:pt-0 xl:min-h-[420px]">
          <div className="md:pb-8 lg:py-12 lg:pb-12">
            <h3 className="mb-5 flex items-center gap-2 text-2xl font-semibold">
              <Replay />
              Bulk Function Replay
            </h3>
            <p className="mb-7 text-lg font-medium text-slate-300">
              Never deal with the hassle of dead-letter-queues. Replay one or <em>millions</em> of
              failed functions at any time with the click of a button.
            </p>

            <a
              href="/docs/platform/replay"
              className="mt-4 font-medium text-slate-200 underline decoration-slate-50/30 decoration-dotted underline-offset-4 hover:text-white hover:decoration-white/50"
            >
              Learn more →
            </a>
          </div>

          <div className="flex h-full flex-col justify-end md:pt-8">
            <img
              src="/assets/homepage/replay-screenshot.png"
              alt="Inngest Dev Server Screenshot"
              className={`
                pointer-events-none
                m-auto -mb-1 w-full max-w-[420px]
                origin-center rounded-md drop-shadow-sm md:min-w-[320px]
                xl:min-w-[380px]
              `}
            />
          </div>
        </div>
      </div>
    </Container>
  );
}
