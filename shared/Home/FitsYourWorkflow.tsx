import CopyBtn from "./CopyBtn";
import Container from "../layout/Container";
import SendEvents from "./HomeImg/SendEvents";
import SectionHeader from "../SectionHeader";
export default function EventDriven() {
  const handleCopyClick = (copy) => {
    navigator.clipboard.writeText(copy);
  };

  return (
    <>
      <Container className="mt-20 mb-12 lg:mr-[564px]">
        <SectionHeader
          title="Fits into your workflow"
          lede="Add Inngest to your stack in a few lines of code, then deploy to your
          existing provider. You don't have to change anything to get started."
        />
      </Container>

      <div className="bg-gradient-to-r from-slate-1000/0 to-slate-900 pb-32 relative z-10">
        <Container className="md:px-20 lg:h-[504px] xl:h-[484px]">
          <div className="py-16 lg:mr-[564px] xl:max-w-lg">
            <h3 className="text-lg xl:text-2xl text-slate-50 mb-3">
              Reliable background functions in one line
            </h3>
            <p className="text-slate-400 text-sm leading-5 lg:leading-7 lg:text-base">
              Use the Inngest SDK to define functions that are triggered by
              events sent from your app (or anywhere on the internet). We call
              your functions by HTTP at the right time, resuming your function
              with the right state &mdash; using normal TypeScript.
            </p>
            <div className="bg-slate-800/50 mt-4 backdrop-blur-md border border-slate-700/30 inline-flex rounded text-sm text-slate-200 shadow-lg">
              <pre className=" pl-4 pr-4 py-2">
                <code className="bg-transparent text-slate-300">
                  <span className="text-cyan-400">npm install</span> inngest
                </code>
              </pre>
              <div className="bg-slate-900/50 rounded-r flex items-center justify-center pl-2 pr-2.5">
                <CopyBtn
                  btnAction={handleCopyClick}
                  copy="npm install inngest"
                />
              </div>
            </div>
          </div>

          <SendEvents />
        </Container>
      </div>

      <Container className="flex flex-col lg:flex-row  gap-6 lg:gap-8 xl:gap-16 -mt-24 ">
        <div className="lg:w-1/2 relative md:mr-40 lg:mr-0">
          <div className="lg:absolute inset-0 rounded-lg bg-blue-500 opacity-20 rotate-2 -z-0 scale-x-[110%] mx-5"></div>
          <div
            style={{
              backgroundImage: "url(/assets/footer/footer-grid.svg)",
              backgroundSize: "cover",
              backgroundPosition: "right -60px top -160px",
              backgroundRepeat: "no-repeat",
            }}
            className=" flex flex-col justify-between text-center bg-blue-500/90 rounded-xl relative w-full h-full"
          >
            <div className=" pt-6 lg:pt-11 px-4 xl:px-16">
              <h4 className="text-white text-xl lg:text-2xl font-medium tracking-tight mb-2">
                Use with your favorite frameworks
              </h4>
              <p className="text-sky-200 text-sm ">
                Write your code directly within your existing codebase.
              </p>
            </div>
            <div className="flex items-center justify-evenly xl:justify-between mt-6 mb-8 flex-wrap m-auto px-8">
              <div className="flex items-end lg:flex-row justify-evenly xl:justify-between w-full m-auto xl:w-1/2">
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-next-js"
                  className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/next-js.png"
                  />
                </a>
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-express"
                  className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/express.png"
                  />
                </a>
              </div>
              <div className="flex items-start lg:flex-row justify-evenly xl:justify-between w-full m-auto xl:w-1/2">
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-redwood"
                  className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/redwood.png"
                  />
                </a>
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-cloudflare"
                  className="flex w-1/2 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/cloudflare-pages.png"
                  />
                </a>
              </div>
            </div>
          </div>
        </div>

        <div className="lg:w-1/2 relative md:ml-40 lg:ml-0">
          <div className="absolute inset-0 rounded-lg bg-purple-500 opacity-20 rotate-2 -z-0 scale-x-[110%] mx-5"></div>
          <div
            style={{
              backgroundImage: "url(/assets/footer/footer-grid.svg)",
              backgroundSize: "cover",
              backgroundPosition: "right -60px top -160px",
              backgroundRepeat: "no-repeat",
            }}
            className=" flex flex-col justify-between text-center bg-purple-500/90 rounded-xl relative w-full h-full"
          >
            <div className=" pt-6 lg:pt-11 px-4 xl:px-16">
              <h4 className="text-white text-xl lg:text-2xl font-medium tracking-tight mb-2">
                Deploy functions anywhere
              </h4>
              <p className="text-purple-100 text-sm ">
                Inngest calls your code, securely, as events are received.
                <br />
                Keep shipping your code as you do today.
              </p>
            </div>
            <div className="flex items-center justify-evenly xl:justify-between mt-6 mb-8 flex-wrap m-auto px-8">
              <div className="flex items-end lg:flex-row justify-evenly xl:justify-between w-full m-auto flex-wrap">
                <a
                  href="/docs/deploy/vercel?ref=homepage-fits-your-workflow"
                  className="flex w-1/2 md:w-1/3 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/vercel.png"
                  />
                </a>
                <a
                  href="/docs/deploy/netlify?ref=homepage-fits-your-workflow"
                  className="flex w-1/2 md:w-1/3 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/netlify.png"
                  />
                </a>
                <a
                  href="/docs/deploy/cloudflare?ref=homepage-fits-your-workflow"
                  className="flex w-1/2 md:w-1/3 max-w-[140px] hover:scale-110 transition-all duration-150"
                >
                  <img
                    className="max-w-[140px]"
                    src="/assets/homepage/send-events/cloudflare-pages.png"
                  />
                </a>
              </div>
            </div>
          </div>
        </div>
      </Container>
    </>
  );
}
