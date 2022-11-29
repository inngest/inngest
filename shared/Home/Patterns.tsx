import HomePatternsCheck from "../Icons/HomePatternsCheck";
import ArrowRight from "../Icons/ArrowRight";
import Container from "./Container";

export default function Patterns() {
  return (
    <Container>
      <div className="absolute top-0 bottom-0 -left-2 -right-2 lg:left-4 lg:right-4 rounded-lg bg-indigo-500 opacity-20 rotate-1 -z-0 mx-5"></div>
      <div
        style={{
          backgroundImage: "url(/assets/footer/footer-grid.svg)",
          backgroundSize: "cover",
          backgroundPosition: "right -60px top -160px",
          backgroundRepeat: "no-repeat",
        }}
        className="mt-20 mb-12 p-8 md:p-12 lg:px-16 lg:py-16 bg-indigo-600 rounded-lg shadow-3xl relative z-10"
      >
        <h3 className="text-slate-50 font-medium text-2xl lg:text-3xl xl:text-4xl mb-4 tracking-tighter ">
          Learn the patterns so you can build anything
        </h3>
        <p className="text-slate-200 font-regular max-w-md lg:max-w-xl text-sm leading-5 lg:leading-6">
          We’ve documented the key patterns that devs encounter when building
          background jobs or scheduled jobs - from the basic to the advanced.
          Read the patterns and learn how to create them with Inngest in just a
          few minutes:
        </p>
        <ul className="flex flex-col gap-1.5 md:gap-0 md:flex-row md:flex-wrap max-w-[600px] mt-6 mb-10">
          <li className="text-slate-200 flex text-sm md:w-1/2 md:mb-2">
            <HomePatternsCheck />{" "}
            <a
              href=""
              className="ml-2 text-slate-200 flex items-bottom group hover:text-white transition-colors"
            >
              Build reliable webhooks
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </li>
          <li className="text-slate-200 flex text-sm md:w-1/2 md:mb-2">
            <HomePatternsCheck />{" "}
            <a
              href=""
              className="ml-2 text-slate-200 flex items-bottom group hover:text-white transition-colors"
            >
              Running functions in parallel
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </li>
          <li className="text-slate-200 flex text-sm md:w-1/2">
            <HomePatternsCheck />{" "}
            <a
              href=""
              className="ml-2 text-slate-200 flex items-bottom group hover:text-white transition-colors"
            >
              Reliably run critical workflows
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </li>
          <li className="text-slate-200 flex text-sm md:w-1/2">
            <HomePatternsCheck />{" "}
            <a
              href=""
              className="ml-2 text-slate-200 flex items-bottom group hover:text-white transition-colors"
            >
              Building flows for lost customers
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
          </li>
        </ul>
        <a
          href="/patterns"
          className="rounded-full inline-flex text-sm font-medium pl-6 pr-5 py-2 bg-slate-800 hover:bg-indigo-800 transition-all text-white gap-1.5 group"
        >
          Browse all patterns
          <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
        </a>
      </div>
    </Container>
  );
}
