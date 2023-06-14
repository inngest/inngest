import Link from "next/link";
import { ChevronRightIcon } from "@heroicons/react/20/solid";

import Container from "../layout/Container";

export default function Hero() {
  return (
    <Container
      className="mt-24 md:mt-36 tracking-tight lg:bg-[url(/assets/homepage/use-case-line-graphic.svg)]"
      style={{
        backgroundSize: "1512px 198px",
        backgroundPosition: "50% 258px",
        backgroundRepeat: "no-repeat",
      }}
    >
      <div className="mb-12 md:mb-32 text-center">
        <h1 className="text-4xl md:text-[3.125rem] md:leading-[3.75rem] font-bold bg-clip-text text-transparent bg-gradient-to-r from-[#E2BEFF] via-white to-[#AFC1FF] drop-shadow">
          Effortless serverless queues,
          <br />
          background jobs, and workflows
        </h1>
        <p className="mt-6 mx-auto max-w-2xl text-lg text-indigo-200 drop-shadow">
          Easily develop serverless workflows in your current codebase, without
          any new infrastructure. Using Inngest, your entire team can ship
          reliable products.
        </p>
      </div>
      <div className="flex flex-col gap-8 pt-12 lg:py-28 items-center justify-center">
        <div>
          <Link
            href="/sign-up?ref=homepage-hero"
            className="rounded-md font-medium px-9 py-3.5 bg-indigo-500 hover:bg-indigo-400 transition-all text-white"
          >
            Start Building For Free
          </Link>
        </div>
        <Link
          href="/docs?ref=homepage-hero"
          className="group flex items-center gap-1 rounded-md pl-3 pr-1.5 py-1.5 bg-transparent transition-all text-indigo-200 border border-transparent hover:border-slate-800 whitespace-nowrap"
        >
          Quick Start Guide{" "}
          <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
        </Link>
      </div>
    </Container>
  );
}
