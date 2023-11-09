import Link from "next/link";
import Container from "../layout/Container";
import clsx from "clsx";

import Heading from "./Heading";
import CopyBtn from "./CopyBtn";
import Replay from "../Icons/Replay";

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
            <br className="hidden md:block" /> Get more done, faster with
            everything built into our platform.
          </>
        }
        className="mx-auto max-w-3xl text-center"
      />

      <div className="my-24 mx-auto max-w-6xl grid grid-cols-1 md:grid-cols-2 gap-8">
        <div className="md:col-span-2 rounded-2xl overflow-hidden xl:min-h-[420px] px-8 pt-12 lg:pt-0 lg:px-16 grid grid-cols-1 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-[40%_1fr] gap-8 md:gap-8 lg:gap-24 items-center bg-gradient-to-br from-emerald-400/10 to-cyan-400/10">
          <div className="md:pb-8 lg:py-12 lg:pb-12">
            <h3 className="text-2xl font-semibold mb-5">Branch environments</h3>
            <p className="text-lg mb-7 font-medium text-slate-300">
              Test your entire application end-to-end with an Inngest
              environment for every development branch that you deploy, without
              any extra work.
            </p>
            <a
              href="/docs/platform/environments"
              className="mt-4 font-medium text-slate-200 hover:text-white hover:underline decoration-dotted underline-offset-4 decoration-slate-50/30"
            >
              Learn more →
            </a>
          </div>

          <div className="flex flex-col justify-end h-full md:pt-8">
            <img
              src="/assets/homepage/branch-envs-screenshot.png"
              alt="Branch environments in the Inngest dashboard"
              className={`
              w-full
                max-w-full md:min-w-[420px] xl:min-w-[520px] -mb-1
                rounded-md drop-shadow-sm m-auto origin-center
                pointer-events-none
                max-w-6xl
              `}
            />
          </div>
        </div>

        <div className="grid grid-rows-[auto_1fr] rounded-lg gap-8 md:gap-10 overflow-hidden bg-gradient-to-b	from-amber-400/10 to-rose-400/15">
          <div className="pt-11 px-8 md:px-10">
            <h3 className="text-2xl font-semibold mb-5">
              Real-time observability metrics
            </h3>
            <p className="text-lg mb-7 font-medium text-slate-300">
              Quickly diagnose system wide issues with built in metrics. View
              backlogs and spikes in failures for every single function. There
              is no need for instrumenting your code for metrics or battling
              some Cloudwatch dashboard.
            </p>
            <a
              href="/blog/2023-10-27-fn-metrics-release"
              className="mt-4 font-medium text-slate-200 hover:text-white hover:underline decoration-dotted underline-offset-4 decoration-slate-50/30"
            >
              Learn more →
            </a>
          </div>
          <div className="flex flex-col justify-end h-full pl-10">
            <img
              src="/assets/homepage/observability-metrics.png"
              alt="Real-time observability metrics"
              className="rounded-tl"
            />
          </div>
        </div>

        <div className="grid grid-rows-[auto_1fr] rounded-lg gap-8 md:gap-10 overflow-hidden bg-gradient-to-br from-blue-400/20 to-orange-200/20">
          <div className="pt-11 px-8 md:px-10">
            <h3 className="text-2xl font-semibold mb-5">Full logs & history</h3>
            <p className="text-lg mb-7 font-medium text-slate-300">
              Inngest keeps a full history of every event and function run
              allowing you to easily debug any production issues. No more
              parsing logs or trying to connect the dots over workflows that
              could span days or weeks.
            </p>
            {/* <a
              href="/blog/2023-10-27-fn-metrics-release"
              className="mt-4 font-medium text-slate-200 hover:text-white hover:underline decoration-dotted underline-offset-4 decoration-slate-50/30"
            >
              Learn more →
            </a> */}
          </div>

          <div className="flex flex-col justify-end h-full pl-10">
            <img
              src="/assets/homepage/screenshot-logs-timeline.png"
              alt="Full logs and history"
              className="rounded-tl"
            />
          </div>
        </div>

        <div className="md:col-span-2 rounded-2xl overflow-hidden xl:min-h-[420px] px-8 pt-12 lg:pt-0 lg:px-16 grid grid-cols-1 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-[40%_1fr] gap-8 md:gap-8 lg:gap-24 items-center bg-gradient-to-bl from-indigo-500/40 to-cyan-200/40">
          <div className="md:pb-8 lg:py-12 lg:pb-12">
            <h3 className="flex gap-2 items-center text-2xl font-semibold mb-5">
              <Replay />
              Bulk Replay
            </h3>
            <p className="text-lg mb-7 font-medium text-slate-300">
              Never deal with the hassle of dead-letter-queues. Replay one or{" "}
              <em>millions</em> of failed functions at any time with the click
              of a button.
            </p>

            <p className="text-xl mb-7 font-bold text-slate-300">
              Coming Q4 2023
            </p>

            {/* <a
              href="/docs/platform/environments"
              className="mt-4 font-medium text-slate-200 hover:text-white hover:underline decoration-dotted underline-offset-4 decoration-slate-50/30"
            >
              Learn more →
            </a> */}
          </div>

          <div className="flex flex-col justify-end h-full md:pt-8">
            <img
              src="/assets/homepage/replay-screenshot.png"
              alt="Inngest Dev Server Screenshot"
              className={`
                w-full
                max-w-[420px] md:min-w-[320px] xl:min-w-[380px] -mb-1
                rounded-md drop-shadow-sm m-auto origin-center
                pointer-events-none
              `}
            />
          </div>
        </div>
      </div>
    </Container>
  );
}
