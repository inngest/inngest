import CopyBtn from "./CopyBtn";
import Container from "../layout/Container";
import SendEvents from "./HomeImg/SendEvents";
import SectionHeader from "../SectionHeader";
import { log } from "console";
import HomePatternsCheck from "../Icons/HomePatternsCheck";
import ArrowRight from "../Icons/ArrowRight";
import Github from "../Icons/Github";

export default function ShipInHours() {
  return (
    <div className="overflow-x-hidden pb-60 -mb-60">
      <div>
        <Container className="mt-6 mb-30 relative z-30">
          <SectionHeader
            center
            pre="Built for every developer"
            title={
              <span className="gap-2 items-end text-slate-200 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter text-center">
                Ship in hours, not weeks
              </span>
            }
          />

          <div className="flex justify-center">
            <p className="text-center max-w-md lg:max-w-xl mt-4 text-slate-200">
            Using Inngest, you can build background jobs, scheduled jobs, and workflows in minutes.  Drop our SDK into your code to get production-grade systems out of the box.
            {/*Build background jobs, scheduled jobs, and workflows by adding our SDK to your existing codebase and redeploying to your current platform.  Think in code without worrying about infra, queues, and config.*/}
            </p>
          </div>

          <div className={`
            grid xl:grid-cols-11 grid-cols-1
            mt-20 mb-6 lg:mb-32
            relative z-10
            bg-slate-900/70 backdrop-blur-sm rounded-xl
          `}>
            {/*
              pt-20 xl:pl-20 px-6 lg:pb-0 pb-8
            */}
            <div className={`
              px-12 pt-20
              xl:col-span-6
              flex flex-col justify-stretch items-center h-full lg:border-r-[1px] border-slate-700
              `}>
              <div className="text-center lg:pb-16 pb-6">
                <p className="font-semibold text-xl mb-4">With Inngest</p>
                <p>Write and deploy workflows as functions — everything else is done for you.</p>
              </div>
              <div className="flex items-center flex-1">
                <img src="/assets/payment-flow.png" alt="With Inngest"  className="max-w-[600px] lg:-mb-[50px] pointer-events-none" />
              </div>
            </div>

            <div className="flex flex-col
              px-12 pt-20
              xl:col-span-5
            ">
              <div className="text-center max-w-[400px] m-auto lg:pb-20 pb-8">
                <p className="font-semibold text-xl mb-4">Without Inngest</p>
                <p>Provision queues, handlers, and glue code for each background job, with state over many jobs.</p>
              </div>
              <div className="flex items-center justify-center flex-1">
                <img src="/assets/without-inngest.svg" alt="Without Inngest" className="lg:max-w-full max-w-full pointer-events-none"/>
              </div>
            </div>
          </div>

          <div className="w-screen max-w-screen relative md:-mt-20 lg:-mt-32 xl:-mt-[600px] xl:mb-[600px] z-20 opacity-50 pointer-events-none relative z-0">
            <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[500px] h-[200px] md:h-[400px] lg:h-[500px] bg-sky-500/20 absolute rounded-full left-1/2 translate-x-[-140%] translate-y-[-70%] "></div>
            <div className=" blur-3xl w-[200px] md:w-[450px] lg:w-[550px] h-[200px] md:h-[450px] lg:h-[550px] bg-indigo-500/30 absolute rounded-full left-1/2 -translate-x-[120%] translate-y-[40%] "></div>
            <div className=" blur-3xl w-[200px] md:w-[300px] lg:w-[400px] h-[200px] md:h-[300px] lg:h-[400px] bg-purple-500/30 absolute rounded-full left-1/2 translate-x-[-230%] translate-y-[40%] "></div>
            <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[500px] h-[200px] md:h-[400px] lg:h-[500px] bg-indigo-500/10 absolute rounded-full bottom-0 left-1/2 translate-x-[200%] -translate-y-[62%] "></div>
            <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[550px] h-[200px] md:h-[400px] lg:h-[550px] bg-purple-500/10 absolute rounded-full bottom-0 left-1/2 translate-x-[250%] translate-y-[90%] "></div>
            <div className=" blur-3xl w-[200px] md:w-[200px] lg:w-[400px] h-[200px] md:h-[200px] lg:h-[400px] bg-blue-500/10 absolute rounded-full bottom-0 left-1/2 translate-x-[50%] translate-y-[6%] "></div>
            <div className="overflow-x-hidden overflow-y-hidden w-screen" />
          </div>

          <div className="grid xl:grid-cols-3 xl:gap-20 gap-y-20 mb-20 lg:grid-cols-1 mt-20 xl:px-32">
            <div>
              <h3 className="font-semibold text-xl mb-4">Focus on functions</h3>
              <p className="text-slate-200">Develop faster by working only on your business logic.  We take care of the hard stuff for you, including retries, concurrency, throttling, rate limiting, and failure replay.</p>
            </div>
            <div>
              <h3 className="font-semibold text-xl mb-4">Simple and powerful primitives</h3>
              <p className="text-slate-200">Write long-running workflows with multiple steps and sleeps as a single function.  Deploy to any platform – even serverless functions.</p>
            </div>
            <div>
              <h3 className="font-semibold text-xl mb-4">Any framework, any platform</h3>
              <p className="text-slate-200">Drop the SDK into your existing codebase and deploy to your current cloud, using your current CI/CD process.</p>
            </div>
          </div>

          <div className="pt-12">
            <p className="text-sm text-center text-gray-400">Works with all the frameworks and platforms you already use:</p>
          </div>

          <div className="flex items-end lg:flex-row justify-evenly xl:justify-center w-full m-auto flex-wrap my-8">
            <a
              href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-next-js"
              className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150 opacity-50 hover:opacity-100"
            >
              <img
                className="max-w-[140px]"
                src="/assets/homepage/send-events/next-js.png"
              />
            </a>
            <a
              href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-express"
              className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150 opacity-50 hover:opacity-100"
            >
              <img
                className="max-w-[140px]"
                src="/assets/homepage/send-events/express.png"
              />
            </a>
            <a
              href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-redwood"
              className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150 opacity-50 hover:opacity-100"
            >
              <img
                className="max-w-[140px]"
                src="/assets/homepage/send-events/redwood.png"
              />
            </a>

            <a
              href="/docs/deploy/vercel?ref=homepage-fits-your-workflow"
              className="flex w-1/2 md:w-1/3 max-w-[140px] hover:scale-110 transition-all duration-150 opacity-50 hover:opacity-100"
            >
              <img
                className="max-w-[140px]"
                src="/assets/homepage/send-events/vercel.png"
              />
            </a>
            <a
              href="/docs/deploy/netlify?ref=homepage-fits-your-workflow"
              className="flex w-1/2 md:w-1/3 max-w-[140px] hover:scale-110 transition-all duration-150 opacity-50 hover:opacity-100"
            >
              <img
                className="max-w-[140px]"
                src="/assets/homepage/send-events/netlify.png"
              />
            </a>
            <a
              href="/docs/deploy/cloudflare?ref=homepage-fits-your-workflow"
              className="flex w-1/2 md:w-1/3 max-w-[140px] hover:scale-110 transition-all duration-150 opacity-50 hover:opacity-100"
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
  )
}
