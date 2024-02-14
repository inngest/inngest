import ArrowRight from '../Icons/ArrowRight';
import SectionHeader from '../SectionHeader';
import Container from '../layout/Container';

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
      <Container className="mb-20 grid grid-flow-row grid-cols-1 grid-rows-4 md:grid-cols-2 md:grid-rows-2 md:gap-x-8 xl:gap-y-20">
        <div>
          <img src="/assets/homepage/out-the-box/trigger-function.jpg" />
          <div className="mt-8 pr-8 lg:px-8">
            <h4 className="mb-2 text-xl text-white lg:text-2xl">Use events to trigger functions</h4>
            <p className="max-w-lg text-sm text-slate-400 lg:text-base">
              Send events from your app, webhooks, or integrations. Use them to trigger one or
              multiple functions.
            </p>
            <a
              href="/docs/quick-start?ref=homepage-everything-you-need"
              className="group mt-4 inline-flex items-center gap-0.5 rounded-full bg-slate-800 py-2 pl-6 pr-5 text-sm  font-medium text-white transition-all hover:bg-slate-700"
            >
              Learn more
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </div>
        </div>
        <div>
          <img src="/assets/homepage/out-the-box/automatic-retry.jpg" />
          <div className="mt-8 pr-8 lg:px-8">
            <h4 className="mb-2 text-xl text-white lg:text-2xl">
              Automatic retries for reliable code
            </h4>
            <p className="max-w-lg text-sm text-slate-400 lg:text-base">
              All functions are retried automatically. Functions can be broken into individual steps
              which are each run independently.
            </p>
            <a
              href="/docs/functions/retries?ref=homepage-everything-you-need"
              className="group mt-4 inline-flex items-center gap-0.5 rounded-full bg-slate-800 py-2 pl-6 pr-5 text-sm  font-medium text-white transition-all hover:bg-slate-700"
            >
              Learn more
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </div>
        </div>
        <div className="mt-8 lg:mt-0">
          <img src="/assets/homepage/out-the-box/sleep.jpg" />
          <div className="mt-8 pr-8 lg:px-8">
            <h4 className="mb-2 text-xl text-white lg:text-2xl">Sleep, schedule, delay</h4>
            <p className="max-w-lg text-sm text-slate-400 lg:text-base">
              Create functions that run over hours, days, or weeks.
            </p>
            <a
              href="/docs/functions/multi-step?ref=homepage-everything-you-need"
              className="group mt-4 inline-flex items-center gap-0.5 rounded-full bg-slate-800 py-2 pl-6 pr-5 text-sm  font-medium text-white transition-all hover:bg-slate-700"
            >
              Learn more
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </div>
        </div>
        <div>
          <img src="/assets/homepage/out-the-box/combine-events.jpg" />
          <div className="mt-8 pr-8 lg:px-8">
            <h4 className="mb-2 text-xl text-white lg:text-2xl">
              Combine events to build powerful flows
            </h4>
            <p className="max-w-lg text-sm text-slate-400 lg:text-base">
              Run code that waits for additional events to create workflows with multiple input
              events like cart abandonment, sales processes, and churn flows.
            </p>
            <a
              href="/docs/functions/multi-step?ref=homepage-everything-you-need"
              className="group mt-4 inline-flex items-center gap-0.5 rounded-full bg-slate-800 py-2 pl-6 pr-5 text-sm  font-medium text-white transition-all hover:bg-slate-700"
            >
              Learn more
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </div>
        </div>
      </Container>
      <Container className="mb-32 flex items-center justify-center">
        <a
          href="/docs?ref=homepage-everything-you-need"
          className="group mt-4 inline-flex items-center gap-0.5 rounded-full bg-indigo-500 py-3 pl-6 pr-5 text-sm  font-medium text-white transition-all hover:bg-indigo-400"
        >
          Learn how to get started
          <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
        </a>
      </Container>
    </>
  );
}
