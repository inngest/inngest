import { log } from 'console';

import ArrowRight from '../Icons/ArrowRight';
import Github from '../Icons/Github';
import HomePatternsCheck from '../Icons/HomePatternsCheck';
import SectionHeader from '../SectionHeader';
import Container from '../layout/Container';
import CopyBtn from './CopyBtn';
import SendEvents from './HomeImg/SendEvents';

export default function ShipInHours() {
  return (
    <div className="-mb-60 overflow-x-hidden pb-60">
      <div>
        <Container className="mb-30 relative z-30 mt-6">
          <SectionHeader
            center
            pre="Built for every developer"
            title={
              <span className="mb-4 items-end gap-2 text-center text-2xl font-medium tracking-tighter text-slate-200 lg:text-4xl xl:text-5xl">
                Ship in hours, not weeks
              </span>
            }
          />

          <div className="flex justify-center">
            <p className="mt-4 max-w-md text-center text-slate-200 lg:max-w-xl">
              Using Inngest, you can build background jobs, scheduled jobs, and workflows in
              minutes. Drop our SDK into your code to get production-grade systems out of the box.
              {/*Build background jobs, scheduled jobs, and workflows by adding our SDK to your existing codebase and redeploying to your current platform.  Think in code without worrying about infra, queues, and config.*/}
            </p>
          </div>

          <div
            className={`
            relative z-10 mb-6
            mt-20 grid grid-cols-1
            rounded-xl bg-slate-900/70
            backdrop-blur-sm lg:mb-32 xl:grid-cols-11
          `}
          >
            {/*
              pt-20 xl:pl-20 px-6 lg:pb-0 pb-8
            */}
            <div
              className={`
              flex h-full
              flex-col
              items-center justify-stretch border-slate-700 px-12 pt-20 lg:border-r-[1px] xl:col-span-6
              `}
            >
              <div className="pb-6 text-center lg:pb-16">
                <p className="mb-4 text-xl font-semibold">With Inngest</p>
                <p>Write and deploy workflows as functions — everything else is done for you.</p>
              </div>
              <div className="flex flex-1 items-center">
                <img
                  src="/assets/payment-flow.png"
                  alt="With Inngest"
                  className="pointer-events-none max-w-[600px] lg:-mb-[50px]"
                />
              </div>
            </div>

            <div
              className="flex flex-col
              px-12 pt-20
              xl:col-span-5
            "
            >
              <div className="m-auto max-w-[400px] pb-8 text-center lg:pb-20">
                <p className="mb-4 text-xl font-semibold">Without Inngest</p>
                <p>
                  Provision queues, handlers, and glue code for each background job, with state over
                  many jobs.
                </p>
              </div>
              <div className="flex flex-1 items-center justify-center">
                <img
                  src="/assets/without-inngest.svg"
                  alt="Without Inngest"
                  className="pointer-events-none max-w-full lg:max-w-full"
                />
              </div>
            </div>
          </div>

          <div className="max-w-screen pointer-events-none relative relative z-0 z-20 w-screen opacity-50 md:-mt-20 lg:-mt-32 xl:-mt-[600px] xl:mb-[600px]">
            <div className=" absolute left-1/2 h-[200px] w-[200px] translate-x-[-140%] translate-y-[-70%] rounded-full bg-sky-500/20 blur-3xl md:h-[400px] md:w-[400px] lg:h-[500px] lg:w-[500px] "></div>
            <div className=" absolute left-1/2 h-[200px] w-[200px] -translate-x-[120%] translate-y-[40%] rounded-full bg-indigo-500/30 blur-3xl md:h-[450px] md:w-[450px] lg:h-[550px] lg:w-[550px] "></div>
            <div className=" absolute left-1/2 h-[200px] w-[200px] translate-x-[-230%] translate-y-[40%] rounded-full bg-purple-500/30 blur-3xl md:h-[300px] md:w-[300px] lg:h-[400px] lg:w-[400px] "></div>
            <div className=" absolute bottom-0 left-1/2 h-[200px] w-[200px] -translate-y-[62%] translate-x-[200%] rounded-full bg-indigo-500/10 blur-3xl md:h-[400px] md:w-[400px] lg:h-[500px] lg:w-[500px] "></div>
            <div className=" absolute bottom-0 left-1/2 h-[200px] w-[200px] translate-x-[250%] translate-y-[90%] rounded-full bg-purple-500/10 blur-3xl md:h-[400px] md:w-[400px] lg:h-[550px] lg:w-[550px] "></div>
            <div className=" absolute bottom-0 left-1/2 h-[200px] w-[200px] translate-x-[50%] translate-y-[6%] rounded-full bg-blue-500/10 blur-3xl md:h-[200px] md:w-[200px] lg:h-[400px] lg:w-[400px] "></div>
            <div className="w-screen overflow-x-hidden overflow-y-hidden" />
          </div>

          <div className="mb-20 mt-20 grid gap-y-20 lg:grid-cols-1 xl:grid-cols-3 xl:gap-20 xl:px-32">
            <div>
              <h3 className="mb-4 text-xl font-semibold">Focus on functions</h3>
              <p className="text-slate-200">
                Develop faster by working only on your business logic. We take care of the hard
                stuff for you, including retries, concurrency, throttling, rate limiting, and
                failure replay.
              </p>
            </div>
            <div>
              <h3 className="mb-4 text-xl font-semibold">Simple and powerful primitives</h3>
              <p className="text-slate-200">
                Write long-running workflows with multiple steps and sleeps as a single function.
                Deploy to any platform – even serverless functions.
              </p>
            </div>
            <div>
              <h3 className="mb-4 text-xl font-semibold">Any framework, any platform</h3>
              <p className="text-slate-200">
                Drop the SDK into your existing codebase and deploy to your current cloud, using
                your current CI/CD process.
              </p>
            </div>
          </div>

          <div className="pt-12">
            <p className="text-center text-sm text-gray-400">
              Works with all the frameworks and platforms you already use:
            </p>
          </div>

          <div className="m-auto my-8 flex w-full flex-wrap items-end justify-evenly lg:flex-row xl:justify-center">
            <a
              href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-next-js"
              className="flex w-1/2 max-w-[140px] opacity-50 transition-all duration-150 hover:scale-110 hover:opacity-100"
            >
              <img className="max-w-[140px]" src="/assets/homepage/send-events/next-js.png" />
            </a>
            <a
              href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-express"
              className="flex w-1/2 max-w-[140px] opacity-50 transition-all duration-150 hover:scale-110 hover:opacity-100"
            >
              <img className="max-w-[140px]" src="/assets/homepage/send-events/express.png" />
            </a>
            <a
              href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-redwood"
              className="flex w-1/2 max-w-[140px] opacity-50 transition-all duration-150 hover:scale-110 hover:opacity-100"
            >
              <img className="max-w-[140px]" src="/assets/homepage/send-events/redwood.png" />
            </a>

            <a
              href="/docs/deploy/vercel?ref=homepage-fits-your-workflow"
              className="flex w-1/2 max-w-[140px] opacity-50 transition-all duration-150 hover:scale-110 hover:opacity-100 md:w-1/3"
            >
              <img className="max-w-[140px]" src="/assets/homepage/send-events/vercel.png" />
            </a>
            <a
              href="/docs/deploy/netlify?ref=homepage-fits-your-workflow"
              className="flex w-1/2 max-w-[140px] opacity-50 transition-all duration-150 hover:scale-110 hover:opacity-100 md:w-1/3"
            >
              <img className="max-w-[140px]" src="/assets/homepage/send-events/netlify.png" />
            </a>
            <a
              href="/docs/deploy/cloudflare?ref=homepage-fits-your-workflow"
              className="flex w-1/2 max-w-[140px] opacity-50 transition-all duration-150 hover:scale-110 hover:opacity-100 md:w-1/3"
            >
              <img
                className="max-w-[140px]"
                src="/assets/homepage/send-events/cloudflare-pages.png"
              />
            </a>
          </div>
        </Container>
      </div>
    </div>
  );
}
