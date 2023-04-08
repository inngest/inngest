import Container from "../layout/Container";
import SectionHeader from "../SectionHeader";
export default function Roadmap() {
  return (
    <>
      <Container className="mt-40">
        <SectionHeader
          title="Inngest Roadmap"
          lede="What we've built and what's up next."
        />
      </Container>

      <Container className="flex flex-col-reverse lg:flex-row gap-2 xl:gap-8 rounded-lg mt-12">
        <div className="w-full lg:w-1/3 ">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Future</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-3 xl:p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Dev server function debugging
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Improved Webhook DX
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Function replay/redrive tools
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Function replay/redrive tools
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Function observability metrics
            </li>
          </ul>
        </div>
        <div className="w-full lg:w-1/3">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Now</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-3 xl:p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Webapp redesign
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Branch Environments w/ Vercel support
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Deploy monitoring
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Function and step-level custom error handling functions
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Function Throttling
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Dev server UI: function lists
            </li>
          </ul>
        </div>
        <div className="w-full lg:w-1/3">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Launched</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-3 xl:p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Scheduled function cancellation{" "}
              <span className="px-1.5 py-1 font-medium leading-none text-white bg-indigo-500 rounded text-xs ml-2">
                New
              </span>
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Function concurrency limits{" "}
              <span className="px-1.5 py-1 font-medium leading-none text-white bg-indigo-500 rounded text-xs ml-2">
                New
              </span>
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Parallel steps
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Step functions
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Typescript support, including generics
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              <div>
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-next-js"
                >
                  Next.js
                </a>
                ,{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-express"
                >
                  Express.js
                </a>
                ,{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-aws-lambda"
                >
                  AWS Lambda
                </a>
                ,{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-redwood"
                >
                  RedwoodJS
                </a>
                ,{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-remix"
                >
                  Remix
                </a>
                , &amp;{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-fresh-deno"
                >
                  Fresh (Deno)
                </a>{" "}
                support
              </div>
              <div className="flex flex-wrap mt-3">
                <span className="bg-cyan-600 text-slate-200 text-xs font-medium leading-none px-2 py-1 rounded-full">
                  Frameworks
                </span>
              </div>
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              <a
                className="text-indigo-400"
                href="/docs/deploy/cloudflare?ref=features-sdk-roadmap"
              >
                Cloudflare Pages
              </a>{" "}
              support
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Inngest local dev server integration
            </li>
          </ul>
        </div>
      </Container>
    </>
  );
}
