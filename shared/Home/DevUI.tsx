import Container from "../layout/Container";
import SectionHeader from "../SectionHeader";
import HomePatternsCheck from "../Icons/HomePatternsCheck";
import ArrowRight from "../Icons/ArrowRight";
import Github from "../Icons/Github";
import CopyBtn from "./CopyBtn";

export default function DevUI() {
  const handleCopyClick = (copy) => {
    navigator.clipboard.writeText(copy);
  };

  return (
    <div className="overflow-hidden pb-60 -mb-60">
      <div>
        <Container className="mt-60 -mb-30 relative z-30">
          <SectionHeader
            title={
              <span className="lg:flex gap-2 items-end text-slate-200 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
                Tools for "lightspeed development*"{" "}
                <span className="inline-block text-sm text-slate-2000 tracking-normal ">
                  *actual words a customer used
                </span>
              </span>
            }
            lede={
              <>
                <p>
                  Our open-source dev server runs on your machine for local testing.
                  Get instant feedback and debugging tools so you can build
                  serverless functions with events like never before.
                </p>
                <div className="flex flex-col md:flex-row items-center justify-start pb-20 pt-8 gap-8">
                  <div className="bg-slate-800/50 backdrop-blur-md border border-slate-700/30 flex rounded text-sm text-slate-200 shadow-lg">
                    <pre className=" pl-4 pr-2 py-2">
                      <code className="bg-transparent text-slate-300">
                        <span className="text-cyan-400">npx</span> inngest-cli
                        dev
                      </code>
                    </pre>
                    <div className="bg-slate-900/50 rounded-r flex items-center justify-center pl-2 pr-2.5">
                      <CopyBtn
                        btnAction={handleCopyClick}
                        copy="npx inngest-cli dev"
                      />
                    </div>
                  </div>
                  <a
                    href="/docs/quick-start?ref=homepage-dev-tools"
                    className="group flex items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
                  >
                    Learn more
                    <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
                  </a>
                </div>
              </>
            }
          />
        </Container>
      </div>

      <div className="w-screen max-w-screen relative md:-mt-20 lg:-mt-32 xl:-mt-32 z-20">
        <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[500px] h-[200px] md:h-[400px] lg:h-[500px] bg-sky-500/20 absolute rounded-full left-1/2 -translate-x-[20%] translate-y-[40%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[450px] lg:w-[550px] h-[200px] md:h-[450px] lg:h-[550px] bg-indigo-500/30 absolute rounded-full left-1/2 -translate-x-[100%] translate-y-[40%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[300px] lg:w-[400px] h-[200px] md:h-[300px] lg:h-[400px] bg-purple-500/30 absolute rounded-full left-1/2 translate-x-[50%] translate-y-[40%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[500px] h-[200px] md:h-[400px] lg:h-[500px] bg-indigo-500/10 absolute rounded-full bottom-0 left-1/2 -translate-x-[20%] -translate-y-[12%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[550px] h-[200px] md:h-[400px] lg:h-[550px] bg-purple-500/10 absolute rounded-full bottom-0 left-1/2 -translate-x-[100%] translate-y-[6%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[200px] lg:w-[400px] h-[200px] md:h-[200px] lg:h-[400px] bg-blue-500/10 absolute rounded-full bottom-0 left-1/2 translate-x-[50%] translate-y-[6%] "></div>
        <div className="overflow-x-hidden overflow-y-hidden w-screen">
          <img
            src="/assets/DevUI.png"
            className="rounded-sm shadow-none m-auto w-screen relative z-10 scale-110 origin-center max-w-[1723px]"
          />
        </div>
      </div>

      <Container className="lg:mb-40">
        <div className="bg-slate-900/70 backdrop-blur-sm -mt-40 pt-40 md:pt-32 lg:px-24 xl:px-32 rounded-xl px-8">
          <h2 className="text-white text-center font-medium pt-4 pb-8 text-2xl">
            Run Inngest locally
          </h2>
          <div className="flex flex-col xl:flex-row justify-between rounded-lg">
            <ul className="text-slate-200 text-sm mr-8">
              <li className="mb-3 flex items-center text-slate-200 xl:max-w-[300px] w-full leading-5">
                <HomePatternsCheck />
                <span className="ml-1.5">Events appear in real-time</span>
              </li>
              <li className="mb-3 flex items-center text-slate-200 xl:max-w-[300px] w-full leading-5">
                <HomePatternsCheck />
                <span className="ml-1.5">View function status at a glance</span>
              </li>
            </ul>
            <ul className="gap-2 mr-8 text-sm">
              <li className="mb-3 flex items-top text-slate-200 xl:max-w-[300px] w-full leading-5 ">
                <HomePatternsCheck />
                <span className="ml-1.5">
                  Inspect event payloads and see exactly what functions are
                  triggered
                </span>
              </li>
              <li className="mb-3 flex items-top text-slate-200 xl:max-w-[300px] w-full leading-5">
                <HomePatternsCheck />
                <span className="ml-1.5">
                  TypeScript types are automatically generated
                </span>
              </li>
            </ul>
            <ul className="gap-2 mr-8 text-sm">
              <li className="mb-3 flex items-top text-slate-200 xl:max-w-[300px] w-full leading-5 ">
                <HomePatternsCheck />
                <span className="ml-1.5">
                  Debug your functions with ease. Check step-by-step output and
                  view logs.
                </span>
              </li>
              <li className="mb-3 flex items-top text-slate-200 xl:max-w-[300px] w-full leading-5 ">
                <HomePatternsCheck />
                <span className="ml-1.5">
                  Re-run functions with one click to quickly iterate and fix
                  bugs.
                </span>
              </li>
            </ul>
          </div>
          <div className="flex items-center justify-center gap-4 pb-16 mt-8 flex-col lg:flex-row">
            <div className="bg-slate-800/50 backdrop-blur-md border border-slate-700/30 flex rounded text-sm text-slate-200 shadow-lg">
              <pre className=" pl-4 pr-4 py-2">
                <code className="bg-transparent text-slate-300">
                  <span className="text-cyan-400">npx</span> inngest-cli dev
                </code>
              </pre>
              <div className="bg-slate-900/50 rounded-r flex items-center justify-center pl-2 pr-2.5">
                <CopyBtn
                  btnAction={handleCopyClick}
                  copy="npx inngest-cli dev"
                />
              </div>
            </div>
            <a
              href="https://github.com/inngest/inngest"
              className="group inline-flex gap-2.5 items-center rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
            >
              View and star <Github />
            </a>
          </div>
        </div>
      </Container>
    </div>
  );
}
