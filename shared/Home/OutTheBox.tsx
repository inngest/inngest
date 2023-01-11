import ArrowRight from "../Icons/ArrowRight";
import Container from "../layout/Container";
import SectionHeader from "../SectionHeader";

export default function OutTheBox() {
  return (
    <>
      <Container className="mt-20 lg:mb-12">
        <SectionHeader
          title="Everything you need - out of the box"
          lede="We built all the features that you need to build powerful applications
          without having to re-invent the wheel."
        />
      </Container>
      <Container className="grid xl:gap-y-20 md:gap-x-8 mb-20 grid-cols-1 grid-rows-4 md:grid-cols-2 md:grid-rows-2 grid-flow-row">
        <div>
          <img src="/assets/homepage/out-the-box/trigger-function.jpg" />
          <div className="pr-8 lg:px-8 mt-8">
            <h4 className="text-white text-xl lg:text-2xl mb-2">
              Use events to trigger functions
            </h4>
            <p className="text-sm lg:text-base text-slate-400 max-w-lg">
              Send events from your app, webhooks, or integrations. Use them to
              trigger one or multiple functions.
            </p>
            <a
              href="/docs/quick-start?ref=homepage-everything-you-need"
              className="group inline-flex mt-4 items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-slate-800 hover:bg-slate-700 transition-all text-white"
            >
              Learn more
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </div>
        </div>
        <div>
          <img src="/assets/homepage/out-the-box/automatic-retry.jpg" />
          <div className="pr-8 lg:px-8 mt-8">
            <h4 className="text-white text-xl lg:text-2xl mb-2">
              Automatic retries for reliable code
            </h4>
            <p className="text-sm lg:text-base text-slate-400 max-w-lg">
              All functions are retried automatically. Functions can be broken
              into individual steps which are each run independently.
            </p>
            <a
              href="/docs/functions/retries?ref=homepage-everything-you-need"
              className="group inline-flex mt-4 items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-slate-800 hover:bg-slate-700 transition-all text-white"
            >
              Learn more
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </div>
        </div>
        <div className="mt-8 lg:mt-0">
          <img src="/assets/homepage/out-the-box/sleep.jpg" />
          <div className="pr-8 lg:px-8 mt-8">
            <h4 className="text-white text-xl lg:text-2xl mb-2">
              Sleep, schedule, delay
            </h4>
            <p className="text-sm lg:text-base text-slate-400 max-w-lg">
              Create functions that run over hours, days, or weeks.
            </p>
            <a
              href="/docs/functions/multi-step?ref=homepage-everything-you-need"
              className="group inline-flex mt-4 items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-slate-800 hover:bg-slate-700 transition-all text-white"
            >
              Learn more
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </div>
        </div>
        <div>
          <img src="/assets/homepage/out-the-box/combine-events.jpg" />
          <div className="pr-8 lg:px-8 mt-8">
            <h4 className="text-white text-xl lg:text-2xl mb-2">
              Combine events to build powerful flows
            </h4>
            <p className="text-sm lg:text-base text-slate-400 max-w-lg">
              Run code that waits for additional events to create workflows with
              multiple input events like cart abandonment, sales processes, and
              churn flows.
            </p>
            <a
              href="/docs/functions/multi-step?ref=homepage-everything-you-need"
              className="group inline-flex mt-4 items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-slate-800 hover:bg-slate-700 transition-all text-white"
            >
              Learn more
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </div>
        </div>
      </Container>
      <Container className="flex items-center justify-center mb-32">
        <a
          href="/docs?ref=homepage-everything-you-need"
          className="group inline-flex mt-4 items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-3  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
        >
          Learn how to get started
          <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
        </a>
      </Container>
    </>
  );
}
