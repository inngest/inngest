import Container from "./Container";
import SectionHeader from "./SectionHeader";
import HomeArrow from "../Icons/HomePatternsCheck";
import ArrowRight from "../Icons/ArrowRight";

export default function DevUI() {
  return (
    <div className="overflow-hidden pb-60 -mb-60">
      <div>
        <Container className="mt-60 -mb-30">
          <SectionHeader
            title={
              <span className="lg:flex gap-2 items-end text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
                Tools for "lightspeed development*"{" "}
                <span className="inline-block text-sm text-slate-500 tracking-normal ">
                  *actual words a customer used
                </span>
              </span>
            }
            lede="Our open-source dev server runs on your machine providing you instant feedback
            and debugging tools so you can build serverless functions with
            events like never before possible."
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
          <div className="flex flex-col xl:flex-row justify-between rounded-lg">
            <ul className="text-slate-50 text-sm mr-8">
              <li className="mb-3 flex items-center text-slate-50 xl:max-w-[300px] w-full leading-5">
                <HomeArrow />
                <span className="ml-1.5">Events appear in real-time</span>
              </li>
              <li className="mb-3 flex items-center text-slate-50 xl:max-w-[300px] w-full leading-5">
                <HomeArrow />
                <span className="ml-1.5">View function status at a glance</span>
              </li>
            </ul>
            <ul className="gap-2 mr-8 text-sm">
              <li className="mb-3 flex items-top text-slate-50 xl:max-w-[300px] w-full leading-5 ">
                <HomeArrow />
                <span className="ml-1.5">
                  Inspect event payloads and see exactly what functions are
                  triggered
                </span>
              </li>
              <li className="mb-3 flex items-top text-slate-50 xl:max-w-[300px] w-full leading-5">
                <HomeArrow />
                <span className="ml-1.5">
                  TypeScript types are automatically generated
                </span>
              </li>
            </ul>
            <ul className="gap-2 mr-8 text-sm">
              <li className="mb-3 flex items-top text-slate-50 xl:max-w-[300px] w-full leading-5 ">
                <HomeArrow />
                <span className="ml-1.5">
                  Debug your functions with ease. Check step-by-step output and
                  view logs.
                </span>
              </li>
              <li className="mb-3 flex items-top text-slate-50 xl:max-w-[300px] w-full leading-5 ">
                <HomeArrow />
                <span className="ml-1.5">
                  Re-run functions with one click to quickly iterate and fix
                  bugs.
                </span>
              </li>
            </ul>
          </div>
          <div className="flex flex-col md:flex-row items-center justify-center pb-20 pt-16 gap-8">
            <pre className="bg-slate-800/50 backdrop-blur-md border border-slate-700/30 rounded px-6 py-2 text-sm text-slate-200 overflow-hidden shadow-lg">
              <code>
                <span className="text-cyan-400">npx</span> inngest-cli dev
              </code>
            </pre>
            <a
              href="/docs/functions?ref=homepage-dev-tools"
              className="group flex items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
            >
              Learn more
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </div>
        </div>
      </Container>
    </div>
  );
}
