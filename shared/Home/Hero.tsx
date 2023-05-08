import Link from "next/link";
import HeroImg from "./HomeImg/HeroImg";
import ArrowRight from "../Icons/ArrowRight";
import Container from "../layout/Container";

export default function Hero() {
  return (
    <div className="relative">
      <Container className="pt-20 pb-16 md:pt-36 md:pb-28 lg:pt-36 lg:pb-40 xl:pt-40 xl:pb-32  2xl:pt-56 2xl:pb-48  flex items-center">
        <HeroImg />
        <div className="max-w-[800px] relative pr-10 lg:px-auto m-x-auto py-10 rounded-lg">
          <h1 className="text-4xl leading-[48px] sm:text-5xl sm:leading-[58px] lg:text-6xl font-semibold lg:leading-[68px] tracking-[-2px] text-slate-50 mb-5 font-extrabold">
            Reliable functions on any platform, in minutes
          </h1>
          <p className="text-sm md:text-base text-slate-300 max-w-xl leading-6 md:leading-7">
            Use TypeScript to write and test <strong>durable functions driven by events</strong> or a schedule.{" "}
            <strong>Deploy to any platform using your current flow</strong> in minutes, zero infrastructure or learning needed.
          </p>

          <ul className="text-sm md:text-base text-slate-300 max-w-xl leading-5 md:leading-6 py-8 grid grid-cols-2 gap-x-4 gap-y-3">
            <li
              className="pl-8"
              style={{
                background: "url(/assets/check-white.svg) no-repeat 0 1px"
              }}
            >Serverless queues</li>
            <li
              className="pl-8"
              style={{
                background: "url(/assets/check-white.svg) no-repeat 0 1px"
              }}
            >Background jobs</li>
            <li
              className="pl-8"
              style={{
                background: "url(/assets/check-white.svg) no-repeat 0 1px"
              }}
            >Scheduled functions</li>
            <li
              className="pl-8"
              style={{
                background: "url(/assets/check-white.svg) no-repeat 0 1px"
              }}
            >Durable functions with memory</li>
            <li
              className="pl-8"
              style={{
                background: "url(/assets/check-white.svg) no-repeat 0 1px"
              }}
            >Complex user-defined workflows</li>
            <li
              className="pl-8"
              style={{
                background: "url(/assets/check-white.svg) no-repeat 0 1px"
              }}
            >AI & LLM chains</li>
          </ul>
          <div className="flex flex-col items-start lg:flex-row gap-4 mt-6 lg:mt-12 lg:items-center">
            <a
              href="sign-up?ref=homepage-hero"
              className="group flex items-center gap-0.5 rounded-full text-sm font-medium pl-6 pr-5 py-2  bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
            >
              Start building for free
              <ArrowRight className="group-hover:translate-x-1.5 relative top-px transition-transform duration-150 " />
            </a>
            <a
              href="/docs?ref=homepage-hero"
              className="rounded-full text-sm font-medium px-6 py-2 bg-slate-800 hover:bg-slate-700 transition-all text-white"
            >
              Read the docs
            </a>
          </div>
        </div>
      </Container>
    </div>
  );
}
