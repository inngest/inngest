import SectionHeader from '../SectionHeader';
import Container from '../layout/Container';
import CopyBtn from './CopyBtn';
import SendEvents from './HomeImg/SendEvents';

export default function EventDriven() {
  const handleCopyClick = (copy) => {
    navigator.clipboard.writeText(copy);
  };

  return (
    <>
      <Container className="relative z-20 mb-12 mt-20 lg:pr-[648px]">
        <SectionHeader
          title="Fits into your workflow"
          lede="Add Inngest to your stack in a few lines of code, then deploy to your
          existing provider. You don't have to change anything to get started."
        />
      </Container>

      <div className="from-slate-1000/0 relative z-40 bg-gradient-to-r to-slate-900 pb-32">
        <Container className="md:px-20 lg:h-[504px] xl:h-[484px]">
          <div className="py-16 lg:mr-[564px] xl:max-w-lg">
            <h3 className="mb-3 text-lg text-slate-50 xl:text-2xl">
              Reliable background functions in one line
            </h3>
            <p className="text-sm leading-5 text-slate-400 lg:text-base lg:leading-7">
              Use the Inngest SDK to define functions that are triggered by events sent from your
              app (or anywhere on the internet). We call your functions by HTTP at the right time,
              resuming your function with the right state &mdash; using normal TypeScript.
            </p>
            <div className="relative z-40 mt-4 inline-flex rounded border border-slate-700/30 bg-slate-800/50 text-sm text-slate-200 shadow-lg backdrop-blur-md">
              <pre className=" py-2 pl-4 pr-4">
                <code className="bg-transparent text-slate-300">
                  <span className="text-cyan-400">npm install</span> inngest
                </code>
              </pre>
              <div className="flex items-center justify-center whitespace-nowrap rounded-r bg-slate-900/50 pl-2 pr-2.5">
                <CopyBtn btnAction={handleCopyClick} copy="npm install inngest" />
              </div>
            </div>
          </div>

          <SendEvents />
        </Container>
      </div>

      <Container className="relative z-50 -mt-24  flex flex-col gap-6 lg:flex-row lg:gap-8 xl:gap-16">
        <div className="relative md:mr-40 lg:mr-0 lg:w-1/2">
          <div className="inset-0 -z-0 mx-5 rotate-2 scale-x-[110%] rounded-lg bg-blue-500 opacity-20 lg:absolute"></div>
          <div
            style={{
              backgroundImage: 'url(/assets/footer/footer-grid.svg)',
              backgroundSize: 'cover',
              backgroundPosition: 'right -60px top -160px',
              backgroundRepeat: 'no-repeat',
            }}
            className=" relative flex h-full w-full flex-col justify-between rounded-xl bg-blue-500/90 text-center"
          >
            <div className=" px-4 pt-6 lg:pt-11 xl:px-16">
              <h4 className="mb-2 text-xl font-medium tracking-tight text-white lg:text-2xl">
                Use with your favorite frameworks
              </h4>
              <p className="text-sm text-sky-200 ">
                Write your code directly within your existing codebase.
              </p>
            </div>
            <div className="m-auto mb-8 mt-6 flex flex-wrap items-center justify-evenly px-8 xl:justify-between">
              <div className="m-auto flex w-full items-end justify-evenly lg:flex-row xl:w-1/2 xl:justify-between">
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-next-js"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110"
                >
                  <img className="max-w-[140px]" src="/assets/homepage/send-events/next-js.png" />
                </a>
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-express"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110"
                >
                  <img className="max-w-[140px]" src="/assets/homepage/send-events/express.png" />
                </a>
              </div>
              <div className="m-auto flex w-full items-start justify-evenly lg:flex-row xl:w-1/2 xl:justify-between">
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-redwood"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110"
                >
                  <img className="max-w-[140px]" src="/assets/homepage/send-events/redwood.png" />
                </a>
                <a
                  href="/docs/sdk/serve?ref=homepage-fits-your-workflow#framework-cloudflare"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110"
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

        <div className="relative md:ml-40 lg:ml-0 lg:w-1/2">
          <div className="absolute inset-0 -z-0 mx-5 rotate-2 scale-x-[110%] rounded-lg bg-purple-500 opacity-20"></div>
          <div
            style={{
              backgroundImage: 'url(/assets/footer/footer-grid.svg)',
              backgroundSize: 'cover',
              backgroundPosition: 'right -60px top -160px',
              backgroundRepeat: 'no-repeat',
            }}
            className=" relative flex h-full w-full flex-col justify-between rounded-xl bg-purple-500/90 text-center"
          >
            <div className=" px-4 pt-6 lg:pt-11 xl:px-16">
              <h4 className="mb-2 text-xl font-medium tracking-tight text-white lg:text-2xl">
                Deploy functions anywhere
              </h4>
              <p className="text-sm text-purple-100 ">
                Inngest calls your code, securely, as events are received.
                <br />
                Keep shipping your code as you do today.
              </p>
            </div>
            <div className="m-auto mb-8 mt-6 flex flex-wrap items-center justify-evenly px-8 xl:justify-between">
              <div className="m-auto flex w-full flex-wrap items-end justify-evenly lg:flex-row xl:justify-between">
                <a
                  href="/docs/deploy/vercel?ref=homepage-fits-your-workflow"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110 md:w-1/3"
                >
                  <img className="max-w-[140px]" src="/assets/homepage/send-events/vercel.png" />
                </a>
                <a
                  href="/docs/deploy/netlify?ref=homepage-fits-your-workflow"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110 md:w-1/3"
                >
                  <img className="max-w-[140px]" src="/assets/homepage/send-events/netlify.png" />
                </a>
                <a
                  href="/docs/deploy/cloudflare?ref=homepage-fits-your-workflow"
                  className="flex w-1/2 max-w-[140px] transition-all duration-150 hover:scale-110 md:w-1/3"
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
