import HomePatternsCheck from 'src/shared/Icons/HomePatternsCheck';

import ArrowRight from '../Icons/ArrowRight';
import SectionHeader from '../SectionHeader';
import Container from '../layout/Container';

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

      <Container className="mt-20 flex flex-col items-start gap-x-8 gap-y-16 md:flex-row lg:gap-16">
        <div className="w-full md:w-1/2">
          <div className="rounded-x mb-8 hidden h-40 w-full bg-slate-950">Image</div>
          <div className="pr-8 lg:px-8">
            <h4 className="mb-2 text-xl text-white lg:text-2xl">
              Full observability at your fingertips
            </h4>
            <p className="max-w-lg text-sm text-slate-400 lg:text-base">
              Our platform surfaces failures so you can fix them faster than ever. You shouldn’t
              spend half your day parsing logs.
            </p>

            <ul className="mt-8 flex flex-col gap-4">
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Full logs - Functions & Events
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Metrics
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Debugging tools
              </li>
            </ul>
          </div>
        </div>
        <div className="w-full md:w-1/2">
          <div className="rounded-x mb-8 hidden h-40 w-full bg-slate-950">Image</div>
          <div className="pr-8 lg:px-8">
            <h4 className="mb-2 text-xl text-white lg:text-2xl">
              We've built the hard stuff for you
            </h4>
            <p className="max-w-lg text-sm text-slate-400 lg:text-base">
              Every feature that you need to run your code reliably, included in every pricing plan.
            </p>
            <ul className="mt-8 flex flex-col gap-4">
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Automatic retries of failed functions
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Event replay
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Function & event versioning
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> TypeScript type generation from events
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Throttling
              </li>
              <li className="flex items-center gap-2 text-sm text-slate-200">
                <HomePatternsCheck /> Idempotency
              </li>
            </ul>
          </div>
        </div>
      </Container>
      <Container className="mb-32 mt-20 flex items-center justify-center">
        <a
          href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=homepage-platform`}
          className="group mt-4 inline-flex items-center gap-0.5 rounded-full bg-indigo-500 py-3 pl-6 pr-5 text-sm  font-medium text-white transition-all hover:bg-indigo-400"
        >
          Sign up for free
          <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
        </a>
      </Container>
    </>
  );
}
