import Container from "./Container";
import SectionHeader from "./SectionHeader";
export default function Roadmap() {
  return (
    <>
      <Container className="mt-40">
        <SectionHeader
          title="Inngest SDK Roadmap"
          lede="What we've built and what's up next."
        />
      </Container>

      <Container className="flex flex-col-reverse lg:flex-row gap-2 xl:gap-8 rounded-lg mt-12">
        <div className="w-full lg:w-1/3 ">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Future</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-3 xl:p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Additional platform support (AWS Lambda, Supabase, Deno)
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Additional framework support (Remix, RedwoodJS)
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Testing APIs
            </li>
          </ul>
        </div>
        <div className="w-full lg:w-1/3">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Now</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-3 xl:p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Step delays, conditional expressions, & event-coordination
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Inngest Cloud deploy
            </li>
          </ul>
        </div>
        <div className="w-full lg:w-1/3">
          <h4 className="text-white text-xl font-medium ml-4 mb-4">Launched</h4>
          <ul className="flex flex-col gap-3 border border-slate-600/10 p-3 xl:p-4 rounded-xl">
            <li className="text-slate-200 bg-slate-800/80 rounded text-sm xl:text-base overflow-hidden">
              <div className="flex items-center px-6 py-4">
                Step functions{" "}
                <span className="px-1.5 py-1 font-medium leading-none text-white bg-indigo-500 rounded text-xs ml-2">
                  New
                </span>
              </div>
              <div className="flex flex-wrap px-4 py-2 bg-slate-900">
                <span className="bg-cyan-600 text-slate-200 text-xs font-medium leading-none px-2 py-1 rounded-full">
                  Frameworks
                </span>
              </div>
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Create event-driven and scheduled functions
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Send events
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              TypeScript: Event Type generation and sync (
              <a
                className="text-indigo-400"
                href="/docs/typescript?ref=features-sdk-roadmap"
              >
                docs
              </a>
              )
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              Typescript support, including generics
            </li>
            <li className="text-slate-200 bg-slate-900 rounded text-sm xl:text-base px-6 py-4">
              <div>
                <a
                  className="text-indigo-400"
                  href="/docs/frameworks/nextjs?ref=features-sdk-roadmap"
                >
                  Next.js
                </a>{" "}
                &amp;{" "}
                <a
                  className="text-indigo-400"
                  href="/docs/frameworks/express?ref=features-sdk-roadmap"
                >
                  Express.js
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
