import SectionHeader from '../SectionHeader';
import Container from '../layout/Container';

export default function Roadmap() {
  return (
    <>
      <Container className="mt-40">
        <SectionHeader title="Inngest Roadmap" lede="What we've built and what's up next." />
      </Container>

      <Container className="mt-12 flex flex-col-reverse gap-2 rounded-lg lg:flex-row xl:gap-8">
        <div className="w-full lg:w-1/3 ">
          <h4 className="mb-4 ml-4 text-xl font-medium text-white">Future</h4>
          <ul className="flex flex-col gap-3 rounded-xl border border-slate-600/10 p-3 xl:p-4">
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Dev server function debugging
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Improved Webhook DX
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Function replay/redrive tools
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Function replay/redrive tools
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Function observability metrics
            </li>
          </ul>
        </div>
        <div className="w-full lg:w-1/3">
          <h4 className="mb-4 ml-4 text-xl font-medium text-white">Now</h4>
          <ul className="flex flex-col gap-3 rounded-xl border border-slate-600/10 p-3 xl:p-4">
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Webapp redesign
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Branch Environments w/ Vercel support
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Deploy monitoring
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Function and step-level custom error handling functions
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Function Throttling
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Dev server UI: function lists
            </li>
          </ul>
        </div>
        <div className="w-full lg:w-1/3">
          <h4 className="mb-4 ml-4 text-xl font-medium text-white">Launched</h4>
          <ul className="flex flex-col gap-3 rounded-xl border border-slate-600/10 p-3 xl:p-4">
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Scheduled function cancellation{' '}
              <span className="ml-2 rounded bg-indigo-500 px-1.5 py-1 text-xs font-medium leading-none text-white">
                New
              </span>
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Function concurrency limits{' '}
              <span className="ml-2 rounded bg-indigo-500 px-1.5 py-1 text-xs font-medium leading-none text-white">
                New
              </span>
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Parallel steps
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Step functions
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Typescript support, including generics
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              <div>
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-next-js"
                >
                  Next.js
                </a>
                ,{' '}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-express"
                >
                  Express.js
                </a>
                ,{' '}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-aws-lambda"
                >
                  AWS Lambda
                </a>
                ,{' '}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-redwood"
                >
                  RedwoodJS
                </a>
                ,{' '}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-remix"
                >
                  Remix
                </a>
                , &amp;{' '}
                <a
                  className="text-indigo-400"
                  href="/docs/sdk/serve?ref=features-sdk-roadmap#framework-fresh-deno"
                >
                  Fresh (Deno)
                </a>{' '}
                support
              </div>
              <div className="mt-3 flex flex-wrap">
                <span className="rounded-full bg-cyan-600 px-2 py-1 text-xs font-medium leading-none text-slate-200">
                  Frameworks
                </span>
              </div>
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              <a
                className="text-indigo-400"
                href="/docs/deploy/cloudflare?ref=features-sdk-roadmap"
              >
                Cloudflare Pages
              </a>{' '}
              support
            </li>
            <li className="rounded bg-slate-900 px-6 py-4 text-sm text-slate-200 xl:text-base">
              Inngest local dev server integration
            </li>
          </ul>
        </div>
      </Container>
    </>
  );
}
