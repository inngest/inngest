import Container from "./Container";
import OutTheBoxTriggerFn from "./HomeImg/OutTheBox/OutTheBoxTriggerFn";
import ScheduledEvents from "./HomeImg/OutTheBox/ScheduledEvents";
import SectionHeader from "./SectionHeader";

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
      <Container className="grid xl:gap-y-20 md:gap-x-8 mb-48 grid-cols-1 grid-rows-4 md:grid-cols-2 md:grid-rows-2 grid-flow-row">
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
          </div>
        </div>
      </Container>
    </>
  );
}
