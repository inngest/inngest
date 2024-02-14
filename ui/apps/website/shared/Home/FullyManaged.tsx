import ArrowRight from "../Icons/ArrowRight";
import Container from "../layout/Container";
import SectionHeader from "../SectionHeader";
import HomePatternsCheck from "src/shared/Icons/HomePatternsCheck";

export default function FullyManaged() {
  return (
    <>
      <Container className="mt-40">
        <SectionHeader
          title="The complete platform, fully managed for you"
          lede="Our serverless platform provides all the observability, tools, and
          features so you can focus on just building your product."
        />
      </Container>

      <Container className="mt-20 flex flex-col md:flex-row items-start gap-x-8 gap-y-16 lg:gap-16">
        <div className="w-full md:w-1/2">
          <div className="w-full h-40 bg-slate-950 rounded-x mb-8 hidden">
            Image
          </div>
          <div className="pr-8 lg:px-8">
            <h4 className="text-white text-xl lg:text-2xl mb-2">
              Full observability at your fingertips
            </h4>
            <p className="text-sm lg:text-base text-slate-400 max-w-lg">
              Our platform surfaces failures so you can fix them faster than
              ever. You shouldn’t spend half your day parsing logs.
            </p>

            <ul className="flex flex-col gap-4 mt-8">
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Full logs - Functions & Events
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Metrics
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Debugging tools
              </li>
            </ul>
          </div>
        </div>
        <div className="w-full md:w-1/2">
          <div className="w-full h-40 bg-slate-950 rounded-x mb-8 hidden">
            Image
          </div>
          <div className="pr-8 lg:px-8">
            <h4 className="text-white text-xl lg:text-2xl mb-2">
              We've built the hard stuff for you
            </h4>
            <p className="text-sm lg:text-base text-slate-400 max-w-lg">
              Every feature that you need to run your code reliably, included in
              every pricing plan.
            </p>
            <ul className="flex flex-col gap-4 mt-8">
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Automatic retries of failed functions
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Event replay
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Function & event versioning
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> TypeScript type generation from events
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Throttling
              </li>
              <li className="flex gap-2 text-slate-200 items-center text-sm">
                <HomePatternsCheck /> Idempotency
              </li>
            </ul>
          </div>
        </div>
      </Container>
      <Container className="flex items-center justify-center mb-32 mt-20">
        <a
          href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=homepage-platform`}
          className="group inline-flex mt-4 items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-3  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
        >
          Sign up for free
          <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
        </a>
      </Container>
    </>
  );
}
