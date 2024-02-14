import ArrowRight from '../Icons/ArrowRight';
import HomePatternsCheck from '../Icons/HomePatternsCheck';
import Container from '../layout/Container';

export default function Patterns() {
  return (
    <Container>
      <div className="absolute -left-2 -right-2 bottom-0 top-0 -z-0 mx-5 rotate-1 rounded-lg bg-indigo-500 opacity-20 lg:left-4 lg:right-4"></div>
      <div
        style={{
          backgroundImage: 'url(/assets/footer/footer-grid.svg)',
          backgroundSize: 'cover',
          backgroundPosition: 'right -60px top -160px',
          backgroundRepeat: 'no-repeat',
        }}
        className="shadow-3xl relative z-10 mb-12 mt-20 rounded-lg bg-indigo-600 p-8 md:p-12 lg:px-16 lg:py-16"
      >
        <h3 className="mb-4 text-2xl font-medium tracking-tighter text-slate-50 lg:text-3xl xl:text-4xl ">
          Learn the patterns so you can build anything
        </h3>
        <p className="font-regular max-w-md text-sm leading-5 text-slate-200 lg:max-w-xl lg:leading-6">
          We’ve documented the key patterns that devs encounter when building background jobs or
          scheduled jobs - from the basic to the advanced. Read the patterns and learn how to create
          them with Inngest in just a few minutes:
        </p>
        <ul className="mb-10 mt-6 flex max-w-[600px] flex-col gap-1.5 md:flex-row md:flex-wrap md:gap-0">
          <li className="flex text-sm text-slate-200 md:mb-2 md:w-1/2">
            <HomePatternsCheck />{' '}
            <a
              href="/patterns/build-reliable-webhooks?ref=homepage-patterns"
              className="items-bottom group ml-2 flex text-slate-200 transition-colors hover:text-white"
            >
              Build reliable webhooks
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </li>
          <li className="flex text-sm text-slate-200 md:mb-2 md:w-1/2">
            <HomePatternsCheck />{' '}
            <a
              href="/patterns/running-functions-in-parallel?ref=homepage-patterns"
              className="items-bottom group ml-2 flex text-slate-200 transition-colors hover:text-white"
            >
              Running functions in parallel
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </li>
          <li className="flex text-sm text-slate-200 md:w-1/2">
            <HomePatternsCheck />{' '}
            <a
              href="/patterns/reliably-run-critical-workflows?ref=homepage-patterns"
              className="items-bottom group ml-2 flex text-slate-200 transition-colors hover:text-white"
            >
              Reliably run critical workflows
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </li>
          <li className="flex text-sm text-slate-200 md:w-1/2">
            <HomePatternsCheck />{' '}
            <a
              href="/patterns/event-coordination-for-lost-customers?ref=homepage-patterns"
              className="items-bottom group ml-2 flex text-slate-200 transition-colors hover:text-white"
            >
              Building flows for lost customers
              <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
            </a>
          </li>
        </ul>
        <a
          href="/patterns?ref=homepage-patterns"
          className="group inline-flex gap-1.5 rounded-full bg-slate-800 py-2 pl-6 pr-5 text-sm font-medium text-white transition-all hover:bg-indigo-800"
        >
          Browse all patterns
          <ArrowRight className="relative top-px transition-transform duration-150 group-hover:translate-x-1.5 " />
        </a>
      </div>
    </Container>
  );
}
